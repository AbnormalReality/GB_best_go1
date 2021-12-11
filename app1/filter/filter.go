package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// IFFilter interface for filter struct&method
type IFFilter interface {
	New(args string, fileCols []string, flgCols []string) (*Filter, error)
	Filter(filerow []string) (string, error)
}

// Parse interface for mock
type Parse interface {
	ParseHeading(fileCols, flgCols []string) (map[string]int, map[string]int)
	ParseCondition(cmd []string, colsMask map[string]int) (*Condition, error)
}

type base string

// operators accepted
const (
	OpE   base = "="
	OpNe  base = "!="
	OpM   base = ">"
	OpL   base = "<"
	OpMe  base = ">="
	OpLe  base = "<="
	OpOr  base = "OR"
	OpAnd base = "AND"
	OpXor base = "XOR"
)

// Condition условие из кс
// column_name OP value [AND/OR column_name OP value] ...
// continent=’Asia’ AND date>’2020-04-14’
type Condition struct {
	Colname    string
	Oper       base
	Value      string
	Nextcond   *Condition
	Nextcondop base
}

// Filter filter object
type Filter struct {
	// query parsed
	Cond *Condition
	// map of cols mask
	ColsMask map[string]int
	// map of cols idx as per original order
	ColsIdx map[string]int
	// iteract via interface
	Parse Parse
}

// New method for create struc via interface IFFilter
func (fl *Filter) New(args string, fileCols []string, flgCols []string) (*Filter, error) {
	return NewFilter(args, fileCols, flgCols, false, nil)
}

// NewFilter инициализация структуры - установление структуры - condition - условия отбора
func NewFilter(args string, fileCols, flgCols []string, mockon bool, ParseIf Parse) (*Filter, error) {
	filter := new(Filter)
	if mockon == false {
		filter.Parse = filter
	} else {
		filter.Parse = ParseIf
	}

	colsMask, colsIdx := filter.Parse.ParseHeading(fileCols, flgCols)

	// cmd := strings.Split(args, " ")

	// special split doesn't split in double quotes
	r := regexp.MustCompile(`[^\s"']+|"[^"]*"`)
	cmd := r.FindAllString(args, -1)

	cond, err := filter.Parse.ParseCondition(cmd, colsMask) //use colsmask just to validate col_name in user query
	if err != nil {
		return nil, err
	}

	//setup Filter object
	return &Filter{
		ColsMask: colsMask,
		ColsIdx:  colsIdx,
		Cond:     cond,
	}, nil
}

// ParseHeading parse flgCols and FileCols make maps for them
// 1)colsMask - map of cols which is user specified in flgCols
// 2)colsIdx - map of original index of columns as per it is in file (to restore original order in output)
func (fl *Filter) ParseHeading(fileCols, flgCols []string) (map[string]int, map[string]int) {

	colsMask := make(map[string]int)
	colsIdx := make(map[string]int)

	for i, col := range fileCols {
		//make key in map Mask
		colsMask[col] = 0
		colsIdx[col] = i
		// if no cols in flag mask all and continue
		if flgCols[0] == "" {
			colsMask[col] = 1
			continue
		}

		for _, flg := range flgCols {
			if col == flg {
				// if match make this key val 1
				colsMask[col] = 1
				break
			}
		}
	}
	return colsMask, colsIdx
}

// ParseCondition -  reads user cmd and creates linked structs Condition
func (fl *Filter) ParseCondition(cmd []string, colsMask map[string]int) (*Condition, error) {
	// parse cli to condition
	// "column_name OP value" Например, age > 40 AND status = “sick”
	cond := new(Condition)

	// first check column_name
	if _, ok := colsMask[cmd[0]]; ok {
		//column is ok
		cond.Colname = cmd[0]
		//now check OP
		switch cmd[1] {
		case "=":
			cond.Oper = OpE
		case "!=":
			cond.Oper = OpNe
		case "<":
			cond.Oper = OpL
		case ">":
			cond.Oper = OpM
		case "<=":
			cond.Oper = OpLe
		case ">=":
			cond.Oper = OpMe
		default:
			err := fmt.Errorf("operand %s not found", cmd[1])
			return nil, err
		}
		// operand ok
		// now check value
		//if we have qoutes remove them
		cond.Value = strings.Trim(cmd[2], "\"")

		// check if we have 4th member AND OR link b/w several conditions
		if len(cmd) > 4 {
			// check link b/w logical sets
			switch cmd[3] {
			case string(OpAnd):
				cond.Nextcondop = OpAnd
			case string(OpOr):
				cond.Nextcondop = OpOr
			case string(OpXor):
				cond.Nextcondop = OpXor
			default:
				err := fmt.Errorf("operand %s not found", cmd[3])
				return nil, err
			}
			//call it recursively till we get to end of all args each time we move by 4 positions in args - to next set
			var err error
			cond.Nextcond, err = fl.ParseCondition(cmd[4:], colsMask)
			if err != nil {
				return nil, err
			}

		}

	} else {
		// create error
		err := fmt.Errorf("column %s not found", cmd[0])
		return nil, err
	}

	//all good return condition struct
	return cond, nil
}

// Check check one logical set like 'lla = 123' OR...
func (fl *Filter) Check(condition *Condition, cols map[string]string) bool {

	if val, ok := cols[condition.Colname]; ok {
		val = strings.Trim(val, "\"")
		switch condition.Oper {
		case OpE:
			if val == condition.Value {
				return true
			}
		case OpNe:
			if val != condition.Value {
				return true
			}
		case OpM:
			if val > condition.Value {
				return true
			}
		case OpL:
			if val < condition.Value {
				return true
			}
		case OpMe:
			if val >= condition.Value {
				return true
			}
		case OpLe:
			if val <= condition.Value {
				return true
			}
		default:
			return false
		}

	}
	// by default return false
	return false
}

// Filter метод структуры - проверка и отбор данных берет мапу Cols и выдает такую же мапу выходных данных
func (fl *Filter) Filter(filerow []string) (string, error) {
	// process data which is
	// map of key - col name: col val in the row
	cols := make(map[string]string)

	for col := range fl.ColsMask {
		//build up row to check against conditions
		cols[col] = filerow[fl.ColsIdx[col]]
	}
	//cols is map of 'col name: col val' in the one current row
	condition := fl.Cond // start
	//initial value for condition check
	var res = false
	var prevRes = false
	var prevNextcondop = ""
	// evaluate through all conditions linked in Cond struct
	for condition != nil {
		res = fl.Check(condition, cols)
		switch prevNextcondop {
		case "AND":
			res = prevRes && res
		case "OR":
			res = prevRes || res
		case "XOR":
			if prevRes != res {
				res = true
			} else {
				res = false
			}
		}
		prevRes = res
		prevNextcondop = string(condition.Nextcondop)
		condition = condition.Nextcond
	}

	if res {
		//condition is true print this cols formatted
		var outarr []string

		// return all row with matched one of the key:val
		// out := cols
		// format output as its in file comma separated val1,val2,... in this row

		for col, val := range cols {
			if fl.ColsMask[col] == 1 {
				outarr = StringSliceIns(outarr, fl.ColsIdx[col], val)
			}
		}

		out := strings.Join(outarr, ",")
		return out, nil
	}

	err := fmt.Errorf("no data within condition found")
	return "", err
}

// StringSliceIns - its a shame go doesnt have this func yet as builtin
// insert element before pos in slice. if pos >= len(arr) insert into tail
func StringSliceIns(arr []string, pos int, elem string) []string {
	if pos < 0 {
		pos = 0
	} else if pos >= len(arr) {
		pos = len(arr)
	}
	out := make([]string, len(arr)+1)
	copy(out[:pos], arr[:pos])
	out[pos] = elem
	copy(out[pos+1:], arr[pos:])
	return out
}