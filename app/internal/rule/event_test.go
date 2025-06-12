package rule

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	v1 "github.com/tianping526/eventbridge/apis/api/eventbridge/service/v1"
)

func TestEventExtGetFieldByPath(t *testing.T) {
	e := EventExt{
		EventExt: &v1.EventExt{
			Event: &v1.Event{
				Id:   123,
				Data: `{"a":1, "b": {"c":2}}`,
			},
		},
	}
	val, _ := e.GetFieldByPath([]string{"id"})
	strID, _ := val.(string)
	id, _ := strconv.ParseUint(strID, 10, 64)
	if !reflect.DeepEqual(id, uint64(123)) {
		t.Fatalf("id expect 123, actaul %d", id)
	}
	val, _ = e.GetFieldByPath([]string{"data", "a"})
	a, _ := val.(float64)
	if !reflect.DeepEqual(a, float64(1)) {
		t.Fatalf("data.a expect 1, actaul %f", a)
	}
	val, _ = e.GetFieldByPath([]string{"data", "b", "c"})
	c, _ := val.(float64)
	if !reflect.DeepEqual(c, float64(2)) {
		t.Fatalf("data.b.c expect 1, actaul %f", c)
	}
	val, _ = e.GetFieldByPath([]string{"data", "b"})
	b, _ := val.(map[string]interface{})
	if !reflect.DeepEqual(b, map[string]interface{}{"c": float64(2)}) {
		t.Fatalf("data.b expect {\"c\":2}, actaul %v", b)
	}
	val, _ = e.GetFieldByPath([]string{"faker"})
	if !IsNotExistsVal(val) {
		t.Fatal("faker should not exist")
	}
	val, _ = e.GetFieldByPath([]string{"data", "f"})
	if !IsNotExistsVal(val) {
		t.Fatal("data.f should not exist")
	}
	val, _ = e.GetFieldByPath([]string{"data", "b", "f"})
	if !IsNotExistsVal(val) {
		t.Fatal("data.b.f should not exist")
	}
	val, _ = e.GetFieldByPath([]string{"faker", "b", "c"})
	if !IsNotExistsVal(val) {
		t.Fatal("faker.b.c should not exist")
	}
	val, _ = e.GetFieldByPath([]string{"data"})
	if !reflect.DeepEqual(val, map[string]interface{}{
		"a": float64(1),
		"b": map[string]interface{}{
			"c": float64(2),
		},
	}) {
		t.Fatal("data should be {\"a\":1, \"b\": {\"c\":2}}")
	}
	e = EventExt{
		EventExt: &v1.EventExt{
			Event: &v1.Event{
				Id:   123,
				Data: `{"a":1, "b": {"c":2}}a`,
			},
		},
	}
	_, err := e.GetFieldByPath([]string{"data"})
	if !IsDataUnmarshalError(err) {
		t.Fatal(err)
	}
	_, err = e.GetFieldByPath([]string{"data", "a"})
	if !IsDataUnmarshalError(err) {
		t.Fatal(err)
	}
	err = fmt.Errorf("wrap err: %w", err)
	if !IsDataUnmarshalError(err) {
		t.Fatal(err)
	}
	err = errors.New("not unmarshal err")
	if IsDataUnmarshalError(err) {
		t.Fatal(err)
	}
}
