package guard_test

import (
	"strings"
	"testing"
	"time"

	"github.com/agubarev/hometown/pkg/util/guard"
	"github.com/gocraft/dbr/v2"
	"github.com/r3labs/diff"
	"github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
	type TestObj struct {
		A         int          `json:"a" editable:"true" db:"a_column"`
		B         float64      `json:"b" db:"b_column"`
		C         string       `json:"c" editable:"true" db:"c_column"`
		Timestamp dbr.NullTime `db:"ts" json:"ts"`
	}

	type args struct {
		obj   interface{}
		names []string
	}
	tests := []struct {
		args    args
		wantErr bool
	}{
		{args{&TestObj{}, []string{"A"}}, false},
		{args{&TestObj{}, []string{"A", "C"}}, false},
		{args{&TestObj{}, []string{"C", "A"}}, false},
		{args{&TestObj{}, []string{"A", "B", "C"}}, true},
		{args{&TestObj{}, []string{"B"}}, true},
		{args{&TestObj{}, []string{"A", "B"}}, true},
		{args{&TestObj{}, []string{"B", "A"}}, true},
		{args{&TestObj{}, []string{"C", "B"}}, true},
		{args{&TestObj{}, []string{"B", "C"}}, true},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args.names, "_"), func(t *testing.T) {
			if err := guard.Check(tt.args.obj, tt.args.names...); (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestList(t *testing.T) {
	a := assert.New(t)

	type TestObj struct {
		A         int          `json:"a" editable:"true" db:"a_column"`
		B         float64      `json:"b" db:"b_column"`
		C         string       `json:"c" editable:"true" db:"c_column"`
		Timestamp dbr.NullTime `db:"ts" json:"ts"`
	}

	obj := new(TestObj)
	editables := guard.ListEditable(obj)

	a.Len(editables, 2)
	a.Equal("A", editables[0])
	a.Equal("C", editables[1])
}

func TestProcureDBChangesFromChangelog(t *testing.T) {
	a := assert.New(t)

	type TestObj struct {
		A         int          `json:"a" editable:"true" db:"a_column"`
		B         float64      `json:"b" db:"b_column"`
		C         string       `json:"c" editable:"true" db:"c_column"`
		Timestamp dbr.NullTime `db:"ts" json:"ts"`
	}

	obj1 := &TestObj{
		A:         1,
		B:         3.14,
		C:         "hello",
		Timestamp: dbr.NewNullTime(time.Now()),
	}

	obj2 := &TestObj{
		A:         2,
		B:         14.3,
		C:         "world",
		Timestamp: dbr.NewNullTime(time.Now().Local().Add(1 * time.Minute)),
	}

	// finding differences
	changelog, err := diff.Diff(obj1, obj2)
	a.NoError(err)
	a.NotNil(changelog)

	// obtaining changes
	changes, err := guard.ProcureDBChangesFromChangelog(obj1, changelog)
	a.NoError(err)
	a.NotNil(changes)
}

func TestDBColumnsFrom(t *testing.T) {
	a := assert.New(t)

	type TestObj struct {
		A         int          `json:"a" editable:"true" db:"a_column"`
		B         float64      `json:"b" db:"b_column"`
		C         string       `json:"c" editable:"true" db:"c_column"`
		Timestamp dbr.NullTime `db:"ts" json:"ts"`
	}

	obj := TestObj{}
	cols := guard.DBColumnsFrom(&obj)
	a.NotEmpty(cols)
}
