package jobs // TODO: Rename to jobs

import (
	"testing"
)

func TestStartMultiple(t *testing.T) {
	jm := JobsManager{}
	id1, err := jm.Start("sleep", "1")
	if err != nil {
		t.Error(err)
	}
	id2, err := jm.Start("sleep", "1")
	if err != nil {
		t.Error(err)
	}
	ok, js := jm.Query(id1)
	if !ok {
		t.Error("Query unable to find ID")
	}
	for (*js).State == Running {
		_, js = jm.Query(id1)
	}
	if (*js).State != Completed {
		t.Error("Job did not complete.")
	}

	ok, js = jm.Query(id2)
	if !ok {
		t.Error("Query unable to find ID")
	}
	for (*js).State == Running {
		_, js = jm.Query(id1)
	}
	if (*js).State != Completed {
		t.Error("Job did not complete.")
	}
}

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

func TestStartQueryOutput(t *testing.T) {
	jm := JobsManager{}
	id, err := jm.Start("echo", "hello")
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
	if string((*js).Output.StdOut) != "hello\n" {
		t.Errorf("Incorrect output expected hello, got %s", (*js).Output.StdOut)
	}
}
func TestStop(t *testing.T) {
	jm := JobsManager{}
	id, err := jm.Start("sleep", "25")
	if err != nil {
		t.Error(err)
	}
	jm.Stop(id)
	ok, js := jm.Query(id)
	if !ok {
		t.Error("Query unable to find ID")
	}
	for (*js).State == Running {
		_, js = jm.Query(id)
	}
	if (*js).State != Stopped {
		t.Error("Job did not show stopped status.")
	}
}
