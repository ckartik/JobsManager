package Jobs // TODO: Rename to jobs

import (
	"testing"
)

func TestStartAndQuery(t *testing.T) {
	jm := JobsManager{}
	id, err := jm.Start("sleep", "1")
	if err != nil {
		t.Error(err)
	}
	ok, js := jm.Query(id)
	if !ok {
		t.Error("Query unable to find ID")
	}
	for (*js).State == Running {
		_, js = jm.Query(id)
	}
	if (*js).State != Completed {
		t.Error("Job did not complete in time.")
	}
}
