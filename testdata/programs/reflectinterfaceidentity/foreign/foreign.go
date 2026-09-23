package foreign

import "reflect"

func Type() reflect.Type { return reflect.TypeFor[interface{ hidden() }]() }
