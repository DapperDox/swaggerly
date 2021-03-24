package models

import (
	"reflect"
	"testing"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testServices = []*Service{
	{
		Hostname: "abc.default.svc.cluster.local",
		Ports:    []*Port{{Name: "http", Port: 80, Protocol: ProtocolHTTP}},
	},
	{
		Hostname: "bob.default.svc.cluster.local",
		Address:  "10.0.1.1",
		Ports:    []*Port{{Name: "https", Port: 443, Protocol: ProtocolHTTPS}},
	},
	{
		Hostname: "cat.default.svc.cluster.local",
		Ports:    []*Port{{Name: "http", Port: 80, Protocol: ProtocolHTTP}},
	},
	{
		Hostname: "dog.default.svc.cluster.local",
		Address:  "10.0.1.2",
		Ports:    []*Port{{Name: "http", Port: 80, Protocol: ProtocolHTTP}, {Name: "grpc", Port: 90, Protocol: ProtocolGRPC}},
	},
	{
		Hostname: "foo.default.svc.cluster.local",
		Ports:    []*Port{{Name: "https", Port: 443, Protocol: ProtocolHTTPS}},
	},
}

var (
	testServiceMap  = NewServiceMap(testServices...)
	emptyServiceMap = NewServiceMap()
)

func TestNewServiceMap(t *testing.T) {
	type args struct {
		items []*Service
	}

	tests := []struct {
		name string
		args args
		want ServiceMap
	}{
		{
			name: "empty map",
			args: args{items: make([]*Service, 0)},
			want: emptyServiceMap,
		},
		{
			name: "map with items",
			args: args{items: testServices},
			want: testServiceMap,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewServiceMap(tt.args.items...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewServiceMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceMap_Insert(t *testing.T) {
	type args struct {
		items []*Service
	}

	tests := []struct {
		name string
		m    ServiceMap
		args args
		want int
	}{
		{
			name: "empty",
			m:    NewServiceMap(),
			args: args{items: make([]*Service, 0)},
			want: 0,
		},
		{
			name: "insert 1 into empty map",
			m:    NewServiceMap(),
			args: args{items: []*Service{testServices[0]}},
			want: 1,
		},
		{
			name: "insert 1 into non-empty map where value already exists",
			m:    NewServiceMap(testServices[0]),
			args: args{items: []*Service{testServices[0]}},
			want: 1,
		},
		{
			name: "insert 1 into non-empty map",
			m:    NewServiceMap(testServices[0]),
			args: args{items: []*Service{testServices[1]}},
			want: 2,
		},
		{
			name: "insert multiple into non-empty map where 1 value already exists",
			m:    NewServiceMap(testServices[0]),
			args: args{items: []*Service{testServices[0], testServices[1]}},
			want: 2,
		},
		{
			name: "insert multiple into non-empty map",
			m:    NewServiceMap(testServices[0]),
			args: args{items: []*Service{testServices[1], testServices[2]}},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.Insert(tt.args.items...)
			if got := tt.m.Len(); got != tt.want {
				t.Errorf("ServiceMap.Insert() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceMap_Delete(t *testing.T) {
	type args struct {
		items []*Service
	}

	tests := []struct {
		name string
		m    ServiceMap
		args args
		want int
	}{
		{
			name: "delete 1 from empty map",
			m:    NewServiceMap(),
			args: args{items: []*Service{testServices[0]}},
			want: 0,
		},
		{
			name: "delete 1 from non-empty map",
			m:    NewServiceMap(testServices[0]),
			args: args{items: []*Service{testServices[0]}},
			want: 0,
		},
		{
			name: "delete multiple from non-empty map where 1 value does not already exist",
			m:    NewServiceMap(testServices[0]),
			args: args{items: []*Service{testServices[0], testServices[1]}},
			want: 0,
		},
		{
			name: "delete multiple from non-empty map",
			m:    NewServiceMap(testServices...),
			args: args{items: []*Service{testServices[1], testServices[2]}},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.Delete(tt.args.items...)
			if got := tt.m.Len(); got != tt.want {
				t.Errorf("ServiceMap.Delete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceMap_Has(t *testing.T) {
	type args struct {
		item *Service
	}

	tests := []struct {
		name string
		m    ServiceMap
		args args
		want bool
	}{
		{
			name: "empty map",
			m:    emptyServiceMap,
			args: args{item: testServices[0]},
			want: false,
		},
		{
			name: "non-empty map without value",
			m:    NewServiceMap(testServices[1]),
			args: args{item: testServices[0]},
			want: false,
		},
		{
			name: "non-empty map with value",
			m:    testServiceMap,
			args: args{item: testServices[3]},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.Has(tt.args.item); got != tt.want {
				t.Errorf("ServiceMap.Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceMap_HasAll(t *testing.T) {
	type args struct {
		items []*Service
	}

	tests := []struct {
		name string
		m    ServiceMap
		args args
		want bool
	}{
		{
			name: "empty map",
			m:    emptyServiceMap,
			args: args{items: []*Service{testServices[0]}},
			want: false,
		},
		{
			name: "non-empty map without value",
			m:    NewServiceMap(testServices[1]),
			args: args{items: []*Service{testServices[0]}},
			want: false,
		},
		{
			name: "non-empty map with all values",
			m:    testServiceMap,
			args: args{items: []*Service{testServices[0], testServices[1]}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.HasAll(tt.args.items...); got != tt.want {
				t.Errorf("ServiceMap.HasAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceMap_HasAny(t *testing.T) {
	type args struct {
		items []*Service
	}

	tests := []struct {
		name string
		m    ServiceMap
		args args
		want bool
	}{
		{
			name: "empty map",
			m:    emptyServiceMap,
			args: args{items: []*Service{testServices[0]}},
			want: false,
		},
		{
			name: "non-empty map without value",
			m:    NewServiceMap(testServices[1]),
			args: args{items: []*Service{testServices[0]}},
			want: false,
		},
		{
			name: "non-empty map with one value",
			m:    NewServiceMap(testServices[1]),
			args: args{items: []*Service{testServices[0], testServices[1]}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.HasAny(tt.args.items...); got != tt.want {
				t.Errorf("ServiceMap.HasAny() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceMap_List(t *testing.T) {
	tests := []struct {
		name string
		m    ServiceMap
		want []*Service
	}{
		{
			name: "empty map",
			m:    emptyServiceMap,
			want: make([]*Service, 0),
		},
		{
			name: "non-empty map with a value",
			m:    NewServiceMap(testServices[1]),
			want: []*Service{testServices[1]},
		},
		{
			name: "non-empty map with many values",
			m:    testServiceMap,
			want: testServices,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.List(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ServiceMap.List() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceMap_Len(t *testing.T) {
	tests := []struct {
		name string
		m    ServiceMap
		want int
	}{
		{
			name: "empty map",
			m:    emptyServiceMap,
			want: 0,
		},
		{
			name: "non-empty map with a value",
			m:    NewServiceMap(testServices[1]),
			want: 1,
		},
		{
			name: "non-empty map with many values",
			m:    testServiceMap,
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.Len(); got != tt.want {
				t.Errorf("ServiceMap.Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

var testDeployments = []*Deployment{
	{
		Name:              "abc-svc",
		Namespace:         "mcmp-rtp",
		CreationTimestamp: v1.NewTime(time.Now()),
		Version:           "1",
	},
	{
		Name:              "bob-svc",
		Namespace:         "mcmp-rtp",
		CreationTimestamp: v1.NewTime(time.Now()),
		Version:           "4",
	},
	{
		Name:              "cat-svc",
		Namespace:         "mcmp-rtp",
		CreationTimestamp: v1.NewTime(time.Now()),
		Version:           "7",
	},
	{
		Name:              "dog-svc",
		Namespace:         "mcmp-rtp",
		CreationTimestamp: v1.NewTime(time.Now()),
		Version:           "3",
	},
	{
		Name:              "foo-svc",
		Namespace:         "mcmp-rtp",
		CreationTimestamp: v1.NewTime(time.Now()),
		Version:           "11",
	},
}

var (
	testDeploymentMap  = NewDeploymentMap(testDeployments...)
	emptyDeploymentMap = NewDeploymentMap()
)

func TestNewDeploymentMap(t *testing.T) {
	type args struct {
		items []*Deployment
	}

	tests := []struct {
		name string
		args args
		want DeploymentMap
	}{
		{
			name: "empty map",
			args: args{items: make([]*Deployment, 0)},
			want: emptyDeploymentMap,
		},
		{
			name: "map with items",
			args: args{items: testDeployments},
			want: testDeploymentMap,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDeploymentMap(tt.args.items...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDeploymentMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentMap_Insert(t *testing.T) {
	type args struct {
		items []*Deployment
	}

	tests := []struct {
		name string
		m    DeploymentMap
		args args
		want int
	}{
		{
			name: "empty",
			m:    NewDeploymentMap(),
			args: args{items: make([]*Deployment, 0)},
			want: 0,
		},
		{
			name: "insert 1 into empty map",
			m:    NewDeploymentMap(),
			args: args{items: []*Deployment{testDeployments[0]}},
			want: 1,
		},
		{
			name: "insert 1 into non-empty map where value already exists",
			m:    NewDeploymentMap(testDeployments[0]),
			args: args{items: []*Deployment{testDeployments[0]}},
			want: 1,
		},
		{
			name: "insert 1 into non-empty map",
			m:    NewDeploymentMap(testDeployments[0]),
			args: args{items: []*Deployment{testDeployments[1]}},
			want: 2,
		},
		{
			name: "insert multiple into non-empty map where 1 value already exists",
			m:    NewDeploymentMap(testDeployments[0]),
			args: args{items: []*Deployment{testDeployments[0], testDeployments[1]}},
			want: 2,
		},
		{
			name: "insert multiple into non-empty map",
			m:    NewDeploymentMap(testDeployments[0]),
			args: args{items: []*Deployment{testDeployments[1], testDeployments[2]}},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.Insert(tt.args.items...)
			if got := tt.m.Len(); got != tt.want {
				t.Errorf("DeploymentMap.Insert() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentMap_Delete(t *testing.T) {
	type args struct {
		items []*Deployment
	}

	tests := []struct {
		name string
		m    DeploymentMap
		args args
		want int
	}{
		{
			name: "delete 1 from empty map",
			m:    NewDeploymentMap(),
			args: args{items: []*Deployment{testDeployments[0]}},
			want: 0,
		},
		{
			name: "delete 1 from non-empty map",
			m:    NewDeploymentMap(testDeployments[0]),
			args: args{items: []*Deployment{testDeployments[0]}},
			want: 0,
		},
		{
			name: "delete multiple from non-empty map where 1 value does not already exist",
			m:    NewDeploymentMap(testDeployments[0]),
			args: args{items: []*Deployment{testDeployments[0], testDeployments[1]}},
			want: 0,
		},
		{
			name: "delete multiple from non-empty map",
			m:    NewDeploymentMap(testDeployments...),
			args: args{items: []*Deployment{testDeployments[1], testDeployments[2]}},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.m.Delete(tt.args.items...)
			if got := tt.m.Len(); got != tt.want {
				t.Errorf("DeploymentMap.Delete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentMap_Has(t *testing.T) {
	type args struct {
		item *Deployment
	}

	tests := []struct {
		name string
		m    DeploymentMap
		args args
		want bool
	}{
		{
			name: "empty map",
			m:    emptyDeploymentMap,
			args: args{item: testDeployments[0]},
			want: false,
		},
		{
			name: "non-empty map without value",
			m:    NewDeploymentMap(testDeployments[1]),
			args: args{item: testDeployments[0]},
			want: false,
		},
		{
			name: "non-empty map with value",
			m:    testDeploymentMap,
			args: args{item: testDeployments[3]},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.Has(tt.args.item); got != tt.want {
				t.Errorf("DeploymentMap.Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentMap_HasAll(t *testing.T) {
	type args struct {
		items []*Deployment
	}

	tests := []struct {
		name string
		m    DeploymentMap
		args args
		want bool
	}{
		{
			name: "empty map",
			m:    emptyDeploymentMap,
			args: args{items: []*Deployment{testDeployments[0]}},
			want: false,
		},
		{
			name: "non-empty map without value",
			m:    NewDeploymentMap(testDeployments[1]),
			args: args{items: []*Deployment{testDeployments[0]}},
			want: false,
		},
		{
			name: "non-empty map with all values",
			m:    testDeploymentMap,
			args: args{items: []*Deployment{testDeployments[0], testDeployments[1]}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.HasAll(tt.args.items...); got != tt.want {
				t.Errorf("DeploymentMap.HasAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentMap_HasAny(t *testing.T) {
	type args struct {
		items []*Deployment
	}

	tests := []struct {
		name string
		m    DeploymentMap
		args args
		want bool
	}{
		{
			name: "empty map",
			m:    emptyDeploymentMap,
			args: args{items: []*Deployment{testDeployments[0]}},
			want: false,
		},
		{
			name: "non-empty map without value",
			m:    NewDeploymentMap(testDeployments[1]),
			args: args{items: []*Deployment{testDeployments[0]}},
			want: false,
		},
		{
			name: "non-empty map with one value",
			m:    NewDeploymentMap(testDeployments[1]),
			args: args{items: []*Deployment{testDeployments[0], testDeployments[1]}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.HasAny(tt.args.items...); got != tt.want {
				t.Errorf("DeploymentMap.HasAny() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentMap_List(t *testing.T) {
	tests := []struct {
		name string
		m    DeploymentMap
		want []*Deployment
	}{
		{
			name: "empty map",
			m:    emptyDeploymentMap,
			want: make([]*Deployment, 0),
		},
		{
			name: "non-empty map with a value",
			m:    NewDeploymentMap(testDeployments[1]),
			want: []*Deployment{testDeployments[1]},
		},
		{
			name: "non-empty map with many values",
			m:    testDeploymentMap,
			want: testDeployments,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.List(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeploymentMap.List() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeploymentMap_Len(t *testing.T) {
	tests := []struct {
		name string
		m    DeploymentMap
		want int
	}{
		{
			name: "empty map",
			m:    emptyDeploymentMap,
			want: 0,
		},
		{
			name: "non-empty map with a value",
			m:    NewDeploymentMap(testDeployments[1]),
			want: 1,
		},
		{
			name: "non-empty map with many values",
			m:    testDeploymentMap,
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.Len(); got != tt.want {
				t.Errorf("DeploymentMap.Len() = %v, want %v", got, tt.want)
			}
		})
	}
}
