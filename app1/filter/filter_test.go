package filter_test

import (
	"github.com/AbnormalReality/GB_best_go1/kurs/app1/filter"
	mocks "github.com/AbnormalReality/GB_best_go1/kurs/app1/filter/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"reflect"
	"strings"
	"testing"
)

func TestFilter_New(t *testing.T) {
	// инициализция мокинг структуры
	ParseMock := &mocks.Parse{}
	// настройки для мок функций
	cond := &filter.Condition{Colname: "a", Oper: filter.OpE, Value: "4.2"}
	// логическое выражение
	args := `a != 4.2 AND b = 3`
	// названия колонок (и названия из файла, как бы)
	line := "a,b,c"
	fileCols := strings.Split(line, ",")
	// process get columns name from cmd
	flgCols := strings.Split(line, ",")
	// структуры для мокинг функций
	colsIdx := map[string]int{
		"a": 0,
		"b": 0,
		"c": 0,
	}

	colsMask := map[string]int{
		"a": 1,
		"b": 1,
		"c": 1,
	}
	// 2 функции которые входят в тестируюмую функцию
	ParseMock.On("ParseHeading", mock.Anything, mock.Anything).Return(colsMask, colsIdx)
	ParseMock.On("ParseCondition", mock.Anything, mock.Anything).Return(cond, nil)

	resfilter, err := filter.NewFilter(args, fileCols, flgCols, true, ParseMock)

	require.NoError(t, err)
	targetFilter := &filter.Filter{
		Cond:     cond,
		ColsMask: colsMask,
		ColsIdx:  colsIdx,
	}

	if !reflect.DeepEqual(resfilter,targetFilter) {
		t.Fatalf("method ParseHeading: expected: %v, got: %v", targetFilter, resfilter)
	}
}

func TestFilter_Check(t *testing.T) {

	args := `a != 4.2 AND b = 3`
	line := "a,b,c"

	fileCols := strings.Split(line, ",")
	// process get columns name from cmd
	flgCols := strings.Split(line, ",")

	filterInit, err := filter.NewFilter(args, fileCols, flgCols, false, nil)
	require.NoError(t, err)

	cond := &filter.Condition{Colname: "a", Oper: filter.OpE, Value: "4.2"}
	// condition "a = 4.2"

	cols := map[string]string{
		"a": "4.2",
		"b": "1",
		"c": "1",
	}

	res := filterInit.Check(cond, cols)

	want := true
	if got := res; got != want {
		t.Errorf("method Check Has = %t, want %t", got, want)
	}

	cond = &filter.Condition{Colname: "b", Oper: filter.OpLe, Value: "4.2"}
	// condition "b <= 4.2"

	cols = map[string]string{
		"a": "4.2",
		"b": "1",
		"c": "1",
	}

	res = filterInit.Check(cond, cols)


	want = true
	if got := res; got != want {
		t.Errorf("method Check Has = %t, want %t", got, want)
	}

}

func TestFilter_Filter(t *testing.T) {

	args := `a != 4.2 AND b = 3`
	line := "a,b,c"

	fileCols := strings.Split(line, ",")
	// process get columns name from cmd
	flgCols := strings.Split(line, ",")

	filter, err := filter.NewFilter(args, fileCols, flgCols, false, nil)

	require.NoError(t, err)

	fileline := `4,5,"wasp"`

	filerow := strings.Split(fileline, ",")

	result, err := filter.Filter(filerow)

	require.Error(t, err)

	//check against file args
	want := ""
	if got := result; got != want {
		t.Errorf("method Filter Has = %s, want %s", got, want)
	}

	fileline = `4,3,"wasp"`

	filerow = strings.Split(fileline, ",")

	result, err = filter.Filter(filerow)

	require.NoError(t, err)

	//check against file args
	want = fileline //same string should be returned
	if got := result; got != want {
		t.Errorf("method Filter Has = %s, want %s", got, want)
	}
}

//////////////////////////////////////// ParseHeading table test
func TestFilter_ParseHeading(t *testing.T) {
	args := `a != 4.2 AND b = 3`
	line := "a,b,c"
	line1 := "a,c"
	fileCols := strings.Split(line, ",")
	// process get columns name from cmd
	flgCols := strings.Split(line1, ",")

	filter, err := filter.NewFilter(args, fileCols, flgCols, false, nil)

	require.NoError(t, err)

	resColsMask, _ := filter.ParseHeading(fileCols,flgCols)

	colsMask := map[string]int{
		"a": 1,
		"b": 0,
		"c": 1,
	}

	if !reflect.DeepEqual(colsMask, resColsMask) {
		t.Fatalf("method ParseHeading: expected: %v, got: %v", colsMask, resColsMask)
	}

	colsIdx := map[string]int{
		"a": 0,
		"b": 1,
		"c": 2,
	}
	_, resColsIdx := filter.ParseHeading(fileCols,flgCols)

	if !reflect.DeepEqual(colsIdx,resColsIdx) {
		t.Fatalf("method ParseHeading: expected: %v, got: %v", colsIdx, resColsIdx)
	}
}

func TestStringSliceIns(t *testing.T) {

	tests := []struct {
		name string
		arr  []string
		pos  int
		elem string
		want []string
	}{
		{"test1 - insert to 1 position",
			[]string{"a", "b", "c"},
			1,
			"xxx",
			[]string{"a", "xxx", "b", "c"},
		},
		{"test2 - insert to tail",
			[]string{"a", "b", "c"},
			100,
			"yyy",
			[]string{"a", "b", "c", "yyy"},
		},
	}

	for _, tc := range tests {
		result := filter.StringSliceIns(tc.arr, tc.pos, tc.elem)
		if !reflect.DeepEqual(tc.want, result) {
			t.Fatalf("%s: expected: %v, got: %v", tc.name, tc.want, result)
		}
	}
}