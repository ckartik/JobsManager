# Design Doc
This design doc lays out the high-level understanding of the 3 key components of the Jobs managment system.

There are 4 sections to this document:
1. Library
2. API
3. CLI
4. CA & Certificate Managment

The **Tradeoffs** will be highlighted at the end of each section.

# Library

The core of the library revolves around a single object called `JobsManager`.
This structure will contain various values inside a JobsInfo Structure which can be indexed via the uuidV4 of the job.

We'll synchronize access to this map with the use of a sync.Map (Discussions around choice can be seen in Tradeoffs section below.)

The main purpose of the library is to handle 4 key functions:
1. Starting a job.
2. Stoping a job.
3. Querying the status of a job.
4. Maintaining and providing logs from a jobs output.

``` go
type JobStatus struct {
	// Options - ["Stopped", "Completed", "Errored"]
	status string
	exitCode uint8
	output JobOutput
}

type JobOutput struct {
	StdOut []byte
	StdErr []byte
}

type JobsInfo struct {
	Command *exec.Cmd
	JobStatus JobStatus
}

// Note: All channels will be buffered with cap 1.
type JobChans struct {
	// Write from API
	Kill chan struct{}{}
	// Read from API
	Status chan JobStatus
}

type JobsManager struct {
	JobInfo sync.Map // uuid -> JobsInfo
	JobChannels sync.Map
}
```

## Start
```go
func (*JobsManager) Start(cmd string, args ...string) (uuid.UUID)
```
There are two major parts to this function.
1. Construct the command object with the parameters provided.
2. Spin up a goroutine, which will send info through status channel when complete.

Finally the function should return the UUIDv4.

## Query
``` go
func (jm, *JobsManager) Query(uuid.UUID) (found bool, js JobStatus)
```
This function will use the uuid to `Load` job status data structure stored inside the jobs Map. If not found in JobsInfo, it will return False. It will detect "active" jobs via a select statments.

```go
// Check channels for value, if exist, load into cache/sync map.
chans := jm.JobChannels.Load(uuid)
select {
	case status <-chans.status
		// Load map with status.
	default:
		continue
}
...
jobInfo = jm.JobInfo.Load(uuid)
if jobInfo.Command == nil {
	return false, nil
}
else if jobInfo.JobStatus == nil {
	return true, JobStatus{status:"Active"}
} else{
	return true, jobInfo.JobStatus
}
```

## Stop
``` go
func (*JobsManager) Stop(uuid.UUID) (killSent bool, error)
```
This function will load the job status state map to see if the job has Exited, if it hasn't it will send a Kill signal using the `os.Process` field.

It will subsequently send a singal through the quit channel.
It will be the responsiblity of the go-routine managing the job to detect the job closed due to a Kill Singal. By using the channel, we can communicate from the Stop function/server request goroutine to alter the controlflow in the worker goroutine, example of control flow is show below.
```go

func (js JobService) Stop(uuid...){
	...
	select {
		case js.jobs.load(uuid).kill <- struct{}{}:
			...
		default:
			...
	}
}
go func(...){
	...
	jobservice.kill = make(chan struct{}{},1)
	...
	if ... // Code Job with an error
	{
		select {
				case <- jobservice.kill:
					// Update status as Kill
				default:
					// Update as errored.
		}
```
## Tradeoffs
There are two key tradeoffs here:
1. The schronization primitive used, there were several options in mind
	1. To have a `RWLock` on a map to allow of synchronized access for all goroutines. The benefit here would be if the process was very Read Heavy on the whole map, which is the case since most clients will tend to read after initial job creation and completion.
	2. To use a `Mutex`, this is the most straightforward approach, but since reads to Map could be a common occurence, it would lead to too much (*hypothesized*) contention.
	3. **To use the `sync.Map` and provided syncrhonized access to disjoint keys in the map. 
	The downside here is we can't enforce type safety, and on every retrieval there is a headache of doing a Type Assertion on the retrieved value.
	The benefit here is that we have good performance with access from goroutines and other concurrent entities (client to main server goroutine) in a very disjoint fashion in regards to keys.**
	_We've decided to go with this approach because our system is expected to have each goroutine that is spun up interact with a single key inside each map, a use-case for which means sync.Map is perfect._
	See excerpt from Go Docs below:
		> The Map type is optimized for two common use cases: (1) when the entry for a given key is only ever written once but read many times, as in caches that only grow, or (2) when multiple goroutines read, write, and overwrite entries for disjoint sets of keys.

2. We decided to store the value of the Exit Code and job completion directly through controlflow of the executing go-routine. We do this rather than re-querying the `[Cmd]` structure because it uses the underlying `os.Process` along with `pid` value to determine job status. This however could be erronous over time since PIDs in the os can cycle. As such the decisions was made to store the job state in the server memory (e.g map structure).

### Scoping
We've scoped the storage of each process to be in heap data-structures. Although this may cause some contention on the heap lock, we see it as reasonable to for the scope of this Project.

# API

There are two key techniques we will be using to wrap the Job Library through the API. 	We'll be using a Middleware service called `go-chi` which will allow us to do our two *tricks* with ease.

1. We will be doing **``dependency enjection``** to ensure our route handlers have access to the requisite Job and Authorization Service. We do this by defining an `[Env]` structure shown below.

	```go
	type Env struct {
		jobService Jobs.JobsManager
		authZService sync.Map // uuid -> list
	}
	```
	We will construct our handlers as follows:
	```go 
	 func (env *Env) createJob(w http.Response...)
	```
	We can subseqeuntly use the go-chi router to pass the handler with the dependent services.
	```go
		r.Post("/", env.createJob)
	```

	This will allow us to interact with the Jobs Manager Service provided by the env.
2. We'll use the **CommonName** that has been verified through our CA chain and the TLS handshake to authenticate user. We will index our `authZService` using the CommonName stored in the client cert. This can be found here: `r.TLS.VerifiedChains[0][0].Subject.CommonName`.
We will make the assumption that the CommonName will always be unique to the client and that CAs will only sign the correct clients.

## Security - Transport
This implementation will be using TLS 1.3 as it's the most recent implementation.
To do this we will set the `MinVersion` field as below.
```go
... &tls.Config{
		...
		MinVersion: tls.VersionTLS13,
}
```

With regards to selection of viable ciphersuites, it seems go doesn't allow you to select specifics. The following is above `CipherSuites []uint16` field in the tls config structure.
> Note that TLS 1.3 ciphersuites are not configurable.

After some research, I came across a blog post by a well-known TLS infrastructure engineer named Joe Shaw, he describes the lack of need to specifiy ciphersuites as a result of the quality of TLS 1.3 ciphersuites. https://www.joeshaw.org/abusing-go-linkname-to-customize-tls13-cipher-suites/

### Benefits
>- The biggest benefit is a speedup, as the handshake only has 5 steps compared to 7 in 1.2.

>Ref: https://kinsta.com/blog/tls-1-3/#:~:text=and%20improved%20security.-,Speed%20Benefits%20of%20TLS%201.3,Time%20(0%2DRTT).
As such we'll ensure that the client and server certificates have all the needed structure to apply to tls 1.3

## Security - Authentication
As we've shown above, we'll use the TLS mutual auth to authenticate the client and use the CommonName as the User Identity.

## Security - Authorization
Our middleware will handle Authorization.
1. When creating a job, it will store the ID of the job into the map with the User ID as the key.
2. When requesting access or mutation of a job, we will have middleware that checks the job ID in the param and ensures the ID matches one that the User has access to.

## Router & Endpoints:

### `/api/job [post]`
This endpoint creates a new Job.

### Payload requirements: 
```graphql
{ "cmd": !String, "args": ![String]}
```

### Response Payload - Success (200): 
```graphql
{ "id": uuidv4}
```

### `/api/job/:jobid [get]`
This endpoint gets details about the job specified at jobid.

This endpoint will use authorization middleware to check the user has access to the endpoint. If the user doesn't have access, it will send a `403` response and stop routing the request.

### Response Payload - Success (200):
```graphql
{ "cmd": !String, "args": ![String], "Status": !String, "exitCode": int}
```

### `/api/job/list [get]`
This endpoint gets details about all jobs under the authed user.

Pagination is outside the scope of this project.

### Response Payload - Success (200): 
```graphql
{ ["cmd": !String, "args": ![String], "Status": !String, "exitCode": int, ...]}
```



### `/api/job/:jobid/output [get]`
This endpoint gets output regarding the job specified for jobid.

This endpoint will use authorization middleware to check the user has access to the endpoint. If the user doesn't have access, it will send a `403` response and stop routing the request.

### Response Payload - Success (200):
```graphql
{
	active: !bool,
	output: {
		stdOut: !String,
		stdErr: !String,
	}
}
```

### `/api/job/:jobid/stop [post]`
This sends a kill signal to the command if still running.

This endpoint will use authorization middleware to check the user has access to the endpoint. If the user doesn't have access, it will send a `403` response and stop routing the request.

### Response Payload - Success: 
```json
{ "status_code": 202 }
```

## Tradeoffs
 - Most of the critical tradeoffs here are around how we handle the use of a self-signed Root Certificate created from the openssl utility.

# Client
The client will have 4 key commands, and will be named `[Jobs]`.
1. Start
2. List
3. Stop
5. Output

```bash
# [] specifies params, ! specifies non-empty constraint
# Command 1
$ Jobs start [!cmd] [arg...]
# CMD Output Happypath
> ID: F08B2593-FE31-4A37-BF64-E1B584AEB255

$ Jobs start [!cmd] [arg...]
> ID: DFB531CF-C112-4A11-8016-3AF8D401D347

# Command 2
$ Jobs list	[id]
>
ID			 							Command			Status
DFB531CF-C112-4A11-8016-3AF8D401D347 	curl			active
F08B2593-FE31-4A37-BF64-E1B584AEB255	wget			active

$ Jobs list F08B2593-FE31-4A37-BF64-E1B584AEB255
>
ID			 							Command			Status
F08B2593-FE31-4A37-BF64-E1B584AEB255	wget			active

# Command 3
$ Jobs stop [!id]
> Stopped # Happypath

```

We will also be storing a list of Job IDs that we recieve from the server inside client memory.

## Tradeoffs
- There aren't many big design decisions here.
- This is mostly because we've decided to handle authentication on the mutual TLS side.
	- Therefore the network level guarantees provided by mTLS handle most of the tradeoffs that would need to be made at this layer.

# CA and Certificate Management & Revocation
- **CA**: We will use the openssl CA and a self-signed server certificate as the Root and **ONLY** CA certificate in the chain. Given the scope of ths project, managing a chain of CA's and an air-gapped root CA is assumed to be outside the scope of this POC.
- **Client Certificate**: We will use the server private key to sign the client certificate. 
	1. We will first generate a CSR with CommonName set to the clients unique ID.
	2. We will sign it oursevles using the server private key to generate a client Certificate for future mTLS.
- **Certificate Revocation**: This is outside the scope of this project as this should only serve as a POC and if any certificates are leaked, we can simply redo the entire self-signed CA process and replace the root (only) CA cert & key manually.

### Server x509 Cert:
```yaml
Certificate:
    Data:
        Version: 1 (0x0)
        Serial Number: 14350400092297812159 (0xc726dcda2e24d8bf)
    Signature Algorithm: sha256WithRSAEncryption
        Issuer: C=CA, ST=ON, L=Brampton, O=Teleport, OU=experiment, CN=localhost/emailAddress=ckrish@live.com
        Validity
            Not Before: Jun  3 16:05:00 2021 GMT
            Not After : Mar 23 16:05:00 2024 GMT
```
### Client x509 Cert:
Note: `Kartik#1` is the commonName and the Identity of the client that will be used to index the authorization map in the server.
```yaml
Certificate:
    Data:
        Version: 1 (0x0)
        Serial Number: 15626978577530676140 (0xd8de2e38925ae3ac)
    Signature Algorithm: sha256WithRSAEncryption
        Issuer: C=CA, ST=ON, L=Brampton, O=Teleport, OU=experiment, CN=localhost/emailAddress=ckrish@live.com
        Validity
            Not Before: Jun  3 16:08:07 2021 GMT
            Not After : Mar 23 16:08:07 2024 GMT
        Subject: C=CA, ST=ON, L=Toronto, O=Kartik Chopra, OU=Kartik, CN=Kartik#1/emailAddress=ckrish@live.com
        Subject Public Key Info:
            Public Key Algorithm: rsaEncryption
                Public-Key: (2048 bit)
```
### Testing purposes
For testing the API, use the following command on curls.
```bash
$ curl -k --cert client.pem --key MyClient1.key https://localhost:8443/
```