// Copyright 2017 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"strings"
	"testing"
)

var validServiceKeys = map[string]struct {
	service Service
}{
	"example-service1.default|grpc,http": {
		service: Service{
			Hostname: "example-service1.default",
			Ports:    []*Port{{Name: "http", Port: 80}, {Name: "grpc", Port: 90}},
		},
	},
	"my-service": {
		service: Service{
			Hostname: "my-service",
			Ports:    []*Port{{Name: "", Port: 80}},
		},
	},
	"svc.ns": {
		service: Service{
			Hostname: "svc.ns",
			Ports:    []*Port{{Name: "", Port: 80}},
		},
	},
	"svc": {
		service: Service{
			Hostname: "svc",
			Ports:    []*Port{{Name: "", Port: 80}},
		},
	},
	"svc|test": {
		service: Service{
			Hostname: "svc",
			Ports:    []*Port{{Name: "test", Port: 80}},
		},
	},
	"svc.default.svc.cluster.local|http-test": {
		service: Service{
			Hostname: "svc.default.svc.cluster.local",
			Ports:    []*Port{{Name: "http-test", Port: 80}},
		},
	},
}

// parseServiceKey is the inverse of the Service.String() method.
func parseServiceKey(s string) (hostname string, ports PortList) {
	parts := strings.Split(s, "|")
	hostname = parts[0]

	var names []string

	if len(parts) > 1 {
		names = strings.Split(parts[1], ",")
	} else {
		names = []string{""}
	}

	for _, name := range names {
		ports = append(ports, &Port{Name: name})
	}

	return
}

func TestServiceString(t *testing.T) {
	for s, svc := range validServiceKeys {
		s1 := serviceKey(svc.service.Hostname, svc.service.Ports)
		if s1 != s {
			t.Errorf("ServiceKey => Got %s, expected %s", s1, s)
		}

		hostname, ports := parseServiceKey(s)

		if hostname != svc.service.Hostname {
			t.Errorf("ParseServiceKey => Got %s, expected %s for %s", hostname, svc.service.Hostname, s)
		}

		if len(ports) != len(svc.service.Ports) {
			t.Errorf("ParseServiceKey => Got %#v, expected %#v for %s", ports, svc.service.Ports, s)
		}
	}
}

func TestServiceExternal(t *testing.T) {
	type args struct {
		svc *Service
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "success - external name empty",
			args: args{svc: &Service{}},
			want: false,
		},
		{
			name: "success - external name populated",
			args: args{svc: &Service{ExternalName: "external-name"}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.svc.External(); got != tt.want {
				t.Errorf("Service.External() = %v, expected = %v", got, tt.want)
			}
		})
	}
}

func TestPortListGet(t *testing.T) {
	p1 := &Port{Name: "testport 1", Port: 8080}
	p2 := &Port{Name: "testport 2", Port: 3000}
	p3 := &Port{Name: "testport 3", Port: 8443}

	plist := PortList{p1, p2, p3}

	type args struct {
		portname string
	}

	tests := []struct {
		name     string
		args     args
		want     *Port
		wantBool bool
	}{
		{
			name:     "success - port exists - testport 1",
			args:     args{portname: "testport 1"},
			want:     p1,
			wantBool: true,
		},
		{
			name:     "success - port exists - testport 2",
			args:     args{portname: "testport 2"},
			want:     p2,
			wantBool: true,
		},
		{
			name:     "success - port exists - testport 3",
			args:     args{portname: "testport 3"},
			want:     p3,
			wantBool: true,
		},
		{
			name:     "success - port does not exist",
			args:     args{portname: "non-existent portname"},
			want:     nil,
			wantBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotbool := plist.get(tt.args.portname)
			if gotbool != tt.wantBool {
				t.Errorf("PortList.get() bool = %v, expected = %v", gotbool, tt.wantBool)

				return
			}

			if got != tt.want {
				t.Errorf("PortList.get() = %v, expected = %v", got, tt.want)
			}
		})
	}
}

func TestHTTPProtocol(t *testing.T) {
	if ProtocolUDP.IsHTTP() {
		t.Errorf("UDP is not HTTP protocol")
	}

	if !ProtocolGRPC.IsHTTP() {
		t.Errorf("gRPC is HTTP protocol")
	}
}

func TestConvertCaseInsensitiveStringToProtocol(t *testing.T) {
	testPairs := []struct {
		name string
		out  Protocol
	}{
		{"tcp", ProtocolTCP},
		{"http", ProtocolHTTP},
		{"HTTP", ProtocolHTTP},
		{"Http", ProtocolHTTP},
		{"https", ProtocolHTTPS},
		{"http2", ProtocolHTTP2},
		{"grpc", ProtocolGRPC},
		{"udp", ProtocolUDP},
		{"Mongo", ProtocolMongo},
		{"mongo", ProtocolMongo},
		{"MONGO", ProtocolMongo},
		{"Redis", ProtocolRedis},
		{"redis", ProtocolRedis},
		{"REDIS", ProtocolRedis},
		{"", ProtocolUnsupported},
		{"SMTP", ProtocolUnsupported},
	}

	for _, testPair := range testPairs {
		out := ConvertCaseInsensitiveStringToProtocol(testPair.name)
		if out != testPair.out {
			t.Errorf("ConvertCaseInsensitiveStringToProtocol(%q) => %q, want %q", testPair.name, out, testPair.out)
		}
	}
}
