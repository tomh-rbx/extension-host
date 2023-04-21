/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package exthost

import (
  "strconv"
)

func ToUInt64(val interface{}) uint64 {
  switch val := val.(type) {
  case int:
    return uint64(val)
  case int32:
    return uint64(val)
  case int64:
    return uint64(val)
  case uint64:
    return val
  case float32:
    return uint64(val)
  case float64:
    return uint64(val)
  case string:
    i, err := strconv.ParseInt(val, 10, 64)
    if err != nil {
      return 0
    }
    return uint64(i)
  default:
    return 0
  }
}

func ToUInt(val interface{}) uint {
  switch val := val.(type) {
  case int:
    return uint(val)
  case int32:
    return uint(val)
  case int64:
    return uint(val)
  case float32:
    return uint(val)
  case float64:
    return uint(val)
  case string:
    i, err := strconv.ParseInt(val, 10, 64)
    if err != nil {
      return 0
    }
    return uint(i)
  default:
    return 0
  }
}

func ToBool(val interface{}) bool {
  if val == nil || val == "" {
    return false
  }
  // parse bool string
  if val, ok := val.(string); ok {
    return val == "true"
  }
  return val.(bool)
}

func ToString(val interface{}) string {
  if val == nil {
    return ""
  }
  switch val := val.(type) {
  case string:
    return val
  case bool:
    if val {
      return "true"
    } else {
      return "false"
    }
  case int:
    return strconv.Itoa(val)
  case int32:
    return strconv.Itoa(int(val))
  case int64:
    return strconv.Itoa(int(val))
  case uint:
    return strconv.Itoa(int(val))
  case uint32:
    return strconv.Itoa(int(val))
  case uint64:
    return strconv.Itoa(int(val))
  case float32:
    return strconv.FormatFloat(float64(val), 'f', -1, 32)
  case float64:
    return strconv.FormatFloat(val, 'f', -1, 64)
  }
  return val.(string)
}

func ToInt64(val interface{}) int64 {
  switch val := val.(type) {
  case int:
    return int64(val)
  case int32:
    return int64(val)
  case int64:
    return val
  case float32:
    return int64(val)
  case float64:
    return int64(val)
  case string:
    i, err := strconv.ParseInt(val, 10, 64)
    if err != nil {
      return 0
    }
    return i
  default:
    return 0
  }
}
