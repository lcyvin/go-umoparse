package utils

import (
	"strconv"
	"strings"
)

func IfaceToString(v interface{}) (string, bool) {
  var out string
  out, ok := v.(string)
  if !ok {
    return "", false
  }

  return out, true
}

func IfaceToFloat(v interface{}) (float64, bool) {
  var out float64
  // first try to directly convert
  out, fOk := v.(float64)
  if fOk {
    return out, true
  }

  str, strOk := v.(string)
  if strOk {
    out, err := strconv.ParseFloat(str, 64)
    if err == nil {
      return out, true
    }
  }
  
  // default/empty
  return 0.0, false
}

func IfaceToInt(v interface{}) (int, bool) {
  var out int
  out, iOk := v.(int)
  if iOk {
    return out, true
  }

  sout, sOk := v.(string)
  if sOk {
    out, err := strconv.Atoi(sout)
    if err == nil {
      return out, true
    }
  }

  return 0, false
}


func IfaceToBool(v interface{}) (bool, bool) {
  b, ok := v.(bool)
  if ok {
    return b, true
  }

  s, ok := v.(string)
  if ok {
    switch strings.ToLower(s) {
    case "1":
    case "true":
    case "yes":
      return true, true
    case "0":
    case "false":
    case "no":
      return false, true
    }
  }

  return false, false
}
