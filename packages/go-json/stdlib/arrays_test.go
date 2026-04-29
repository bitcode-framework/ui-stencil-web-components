package stdlib

import (
	"fmt"
	"testing"
)

func TestAppend_ToEmpty(t *testing.T) {
	env := map[string]any{"arr": []any{}}
	result := evalExpr(t, "append(arr, 1)", env)
	arr := result.([]any)
	if len(arr) != 1 || toF64(arr[0]) != 1 {
		t.Errorf("expected [1], got %v", arr)
	}
}

func TestAppend_ToExisting(t *testing.T) {
	env := map[string]any{"arr": []any{1, 2}}
	result := evalExpr(t, "append(arr, 3)", env)
	arr := result.([]any)
	if len(arr) != 3 {
		t.Errorf("expected length 3, got %d", len(arr))
	}
	if toF64(arr[0]) != 1 || toF64(arr[1]) != 2 || toF64(arr[2]) != 3 {
		t.Errorf("expected [1,2,3], got %v", arr)
	}
}

func TestAppend_String(t *testing.T) {
	env := map[string]any{"arr": []any{"a", "b"}}
	result := evalExpr(t, `append(arr, "c")`, env)
	arr := result.([]any)
	if len(arr) != 3 || arr[2] != "c" {
		t.Errorf("expected [a,b,c], got %v", arr)
	}
}

func TestPrepend_ToExisting(t *testing.T) {
	env := map[string]any{"arr": []any{2, 3}}
	result := evalExpr(t, "prepend(arr, 1)", env)
	arr := result.([]any)
	if len(arr) != 3 {
		t.Errorf("expected length 3, got %d", len(arr))
	}
	if toF64(arr[0]) != 1 || toF64(arr[1]) != 2 || toF64(arr[2]) != 3 {
		t.Errorf("expected [1,2,3], got %v", arr)
	}
}

func TestPrepend_ToEmpty(t *testing.T) {
	env := map[string]any{"arr": []any{}}
	result := evalExpr(t, "prepend(arr, 1)", env)
	arr := result.([]any)
	if len(arr) != 1 || toF64(arr[0]) != 1 {
		t.Errorf("expected [1], got %v", arr)
	}
}

func TestSlice_Middle(t *testing.T) {
	env := map[string]any{"arr": []any{1, 2, 3, 4, 5}}
	result := evalExpr(t, "slice(arr, 1, 3)", env)
	arr := result.([]any)
	if len(arr) != 2 {
		t.Errorf("expected length 2, got %d", len(arr))
	}
	if toF64(arr[0]) != 2 || toF64(arr[1]) != 3 {
		t.Errorf("expected [2,3], got %v", arr)
	}
}

func TestSlice_OutOfBounds(t *testing.T) {
	env := map[string]any{"arr": []any{1, 2, 3}}
	result := evalExpr(t, "slice(arr, 5, 10)", env)
	arr := result.([]any)
	if len(arr) != 0 {
		t.Errorf("expected empty array, got %v", arr)
	}
}

func TestSlice_FromStart(t *testing.T) {
	env := map[string]any{"arr": []any{1, 2, 3, 4, 5}}
	result := evalExpr(t, "slice(arr, 0, 2)", env)
	arr := result.([]any)
	if len(arr) != 2 || toF64(arr[0]) != 1 || toF64(arr[1]) != 2 {
		t.Errorf("expected [1,2], got %v", arr)
	}
}

func TestSlice_NoEnd(t *testing.T) {
	env := map[string]any{"arr": []any{1, 2, 3, 4, 5}}
	result := evalExpr(t, "slice(arr, 3)", env)
	arr := result.([]any)
	if len(arr) != 2 || toF64(arr[0]) != 4 || toF64(arr[1]) != 5 {
		t.Errorf("expected [4,5], got %v", arr)
	}
}

func TestChunk_Basic(t *testing.T) {
	env := map[string]any{"arr": []any{1, 2, 3, 4, 5}}
	result := evalExpr(t, "chunk(arr, 2)", env)
	chunks := result.([]any)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	c0 := chunks[0].([]any)
	c1 := chunks[1].([]any)
	c2 := chunks[2].([]any)
	if len(c0) != 2 || len(c1) != 2 || len(c2) != 1 {
		t.Errorf("expected chunk sizes [2,2,1], got [%d,%d,%d]", len(c0), len(c1), len(c2))
	}
	if toF64(c2[0]) != 5 {
		t.Errorf("expected last chunk [5], got %v", c2)
	}
}

func TestChunk_Empty(t *testing.T) {
	env := map[string]any{"arr": []any{}}
	result := evalExpr(t, "chunk(arr, 2)", env)
	chunks := result.([]any)
	if len(chunks) != 0 {
		t.Errorf("expected empty result, got %v", chunks)
	}
}

func TestChunk_ExactDivision(t *testing.T) {
	env := map[string]any{"arr": []any{1, 2, 3, 4}}
	result := evalExpr(t, "chunk(arr, 2)", env)
	chunks := result.([]any)
	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(chunks))
	}
}

func TestZip_Basic(t *testing.T) {
	env := map[string]any{
		"a": []any{1, 2, 3},
		"b": []any{"a", "b"},
	}
	result := evalExpr(t, "zip(a, b)", env)
	zipped := result.([]any)
	if len(zipped) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(zipped))
	}
	pair0 := zipped[0].([]any)
	pair1 := zipped[1].([]any)
	if toF64(pair0[0]) != 1 || pair0[1] != "a" {
		t.Errorf("expected [1, 'a'], got %v", pair0)
	}
	if toF64(pair1[0]) != 2 || pair1[1] != "b" {
		t.Errorf("expected [2, 'b'], got %v", pair1)
	}
}

func TestZip_EqualLength(t *testing.T) {
	env := map[string]any{
		"a": []any{1, 2},
		"b": []any{"x", "y"},
	}
	result := evalExpr(t, "zip(a, b)", env)
	zipped := result.([]any)
	if len(zipped) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(zipped))
	}
}

func TestZip_Empty(t *testing.T) {
	env := map[string]any{
		"a": []any{},
		"b": []any{1, 2},
	}
	result := evalExpr(t, "zip(a, b)", env)
	zipped := result.([]any)
	if len(zipped) != 0 {
		t.Errorf("expected empty result, got %v", zipped)
	}
}

func TestAppend_DoesNotMutateOriginal(t *testing.T) {
	original := []any{1, 2}
	env := map[string]any{"arr": original}
	result := evalExpr(t, "append(arr, 3)", env)
	arr := result.([]any)
	if len(arr) != 3 {
		t.Errorf("expected new array length 3, got %d", len(arr))
	}
	if len(original) != 2 {
		t.Errorf("original array was mutated: expected length 2, got %d", len(original))
	}
}

func TestPrepend_DoesNotMutateOriginal(t *testing.T) {
	original := []any{2, 3}
	env := map[string]any{"arr": original}
	result := evalExpr(t, "prepend(arr, 1)", env)
	arr := result.([]any)
	if len(arr) != 3 {
		t.Errorf("expected new array length 3, got %d", len(arr))
	}
	if len(original) != 2 {
		t.Errorf("original array was mutated: expected length 2, got %d", len(original))
	}
}

func arrToStr(arr []any) string {
	return fmt.Sprintf("%v", arr)
}
