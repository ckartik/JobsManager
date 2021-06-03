# Design Doc
This design doc lays out the high-level understanding of the 3 key components of the Jobs managment system.

There are 4 sections to this document:
1. Library
2. API
3. CLI

The **Tradeoffs** will be highlighted at the end of each section.

# Library

The core of the library revolves around a single object called `JobsManager`. This structure will contain various Sync Maps each of which can be indexed via the uuidV4 of the job to retrieve job information.

The main purpose of the library is to handle 4 key functions:
1. Starting a job.
2. Stoping a job.
3. Querying the status of a job.
4. Maintaining and providing logs from a jobs output.

``` go
type JobsManager struct {
	Jobs sync.Map
	Output sync.Map
}
```

## Start
```go
func (*JobsManager) Start(string, ...string) (uuid.UUID, bool)
```
There are two major parts to this function.
1. Construct the command object with the parameters provided.
2. Spin up a goroutine, where the output is hooked into the `[Output]` sync map.

Finally the function should return the UUIDv4 and a boolean signature notifying success of job creation.

## Query
``` go
func (*JobsManager) Query(uuid.UUID) (exited bool, exitCode int, error)
```
This function will use the uuid to `Load` the correct `*exec.Cmd` value and check the underlying `os.ProcessState` to see if the job has Exited and what the exit status is.

## Stop
``` go
func (*JobsManager) Stop(uuid.UUID) (bool, error)
```
This function will use the uuid to `Load` the correct `*exec.Cmd` value and check the underlying `os.ProcessState` to see if the job has Exited, if it hasn't it will send a Kill signal using the `os.Process` field.

## Tradeoffs
There are two key tradeoffs here:
1. The schronization primitive used, there were several options in mind
	1. To have a `RWLock` on a map to allow of synchronized access for all goroutines. The benefit here would be if the process was very Read Heavy on the whole map.
	2. To use a `Mutex`, this is the most straightforward approach, but since reads to Map could be a common occurence, it would lead to too *hypothesized* much contention.
	3. To use the `sync.Map` and provided syncrhonized access to disjoint keys in the map. The downside here is we can't enforce type safety, and on every retrieval there is a headache of doing a Type Assertion on the retrieved value.

# API

There are two key techniques we will be using to wrap the Job Library through the API. 	We'll be using a Middleware service called `go-chi` which will allow us to do our two *tricks* with ease.
1. We will be doing **``dependency enjection``** to ensuring our route handlers have access to the requisite Job and Authorization Service. We do this by defining an `[Env]` structure shown below.

	```go
	type Env struct {
		jobService Jobs.JobsManager
		authZService sync.Map
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

	This will allow us to interact with the Jobs Manager Service provided by the 
2. We'll use the CommonName that has been verified through our CA chain and the TLS handshake to authenticate user. We will index our `authZService` using the CommonName stored in the client cert. This can be found here: `r.TLS.VerifiedChains[0][0].Subject.CommonName`.
We will make the assumption that the CommonName will always be unique to the client and that CAs will only sign the correct clients.

### Testing purposes
For testing the API, use the following command on curls.
```bash
$ curl -k --cert client.pem --key MyClient1.key https://localhost:8443/
```