package lib

import(
  "reflect"
  "errors"
  "fmt"
)

func SetIntField(c interface{}, field string, value int64) {
  v := reflect.Indirect(reflect.ValueOf(c)).Elem()
  if v.Kind() == reflect.Ptr {
    v = v.Elem()
  }
  v = v.FieldByName(field)
  if v.IsValid() {
    if v.CanSet() {
      if v.Kind() == reflect.Int {
        v.SetInt(value)
      }
    }
  }
}

func SetStringField(c interface{}, field string, value string) {
  v := reflect.Indirect(reflect.ValueOf(c)).Elem()
  if v.Kind() == reflect.Ptr {
    v = v.Elem()
  }
  v = v.FieldByName(field)
  if v.IsValid() {
    if v.CanSet() {
      if v.Kind() == reflect.String {
        v.SetString(value)
      }
    }
  }
}

func SetBoolField(c interface{}, field string, value bool) {
  v := reflect.Indirect(reflect.ValueOf(c)).Elem()
  if v.Kind() == reflect.Ptr {
    v = v.Elem()
  }
  v = v.FieldByName(field)
  if v.IsValid() {
    if v.CanSet() {
      if v.Kind() == reflect.Bool {
        v.SetBool(value)
      }
    }
  }
}

func AppendToSlice(c interface{}, field string, value string) {
  v := reflect.Indirect(reflect.ValueOf(c)).Elem()
  if v.Kind() == reflect.Ptr {
    v = v.Elem()
  }
  v = v.FieldByName(field)
  if v.IsValid() {
    if v.CanSet() {
      if v.Kind() == reflect.Slice {
        v.Set(reflect.Append(v, reflect.ValueOf(value)))
      }
    }
  }
}

func GetFieldType(c interface{}, field string) (string, error) {
  //v := reflect.ValueOf(c)
  v := reflect.Indirect(reflect.ValueOf(c)).Elem()
  if v.Kind() == reflect.Ptr {
    v = v.Elem()
  }
  v = v.FieldByName(field)
  if v.IsValid() {
    return v.Type().String(), nil
  } else {
    return "", errors.New(fmt.Sprintf("Field '%s' does not exist", field))
  }
}