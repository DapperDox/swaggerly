package discover

import (
	"net/url"
	"testing"

	"github.com/go-openapi/swag"

	"github.com/kenjones-cisco/dapperdox/config"
)

func Test_apiLoader_load(t *testing.T) {
	_ = config.LoadFixture("../fixtures")

	type fields struct {
		hostoverride *string
		spec         string
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "fail - location not set",
			fields: fields{
				hostoverride: swag.String(""),
				spec:         "",
			},
			wantErr: true,
		},
		{
			name: "fail - location not valid",
			fields: fields{
				hostoverride: swag.String("example.com:8080"),
				spec:         "",
			},
			wantErr: true,
		},
		{
			name: "fail - unable to read fixture - 500 response code",
			fields: fields{
				hostoverride: nil,
				spec:         "fixtures/fake-spec-does-not-exist.json",
			},
			wantErr: true,
		},
		{
			name: "fail - load bad swagger api spec",
			fields: fields{
				hostoverride: nil,
				spec:         "fixtures/bad_api.json",
			},
			wantErr: true,
		},
		{
			name: "success - loads petstore",
			fields: fields{
				hostoverride: nil,
				spec:         "fixtures/petstore_api.json",
			},
			wantErr: false,
		},
		{
			name: "success - loads iam",
			fields: fields{
				hostoverride: nil,
				spec:         "fixtures/iam_api.json",
			},
			wantErr: false,
		},
		{
			name: "success - loads approvals",
			fields: fields{
				hostoverride: nil,
				spec:         "fixtures/approvals_api.json",
			},
			wantErr: false,
		},
		{
			name: "success - loads aws",
			fields: fields{
				hostoverride: nil,
				spec:         "fixtures/aws_api.json",
			},
			wantErr: false,
		},
		{
			name: "success - loads openstack",
			fields: fields{
				hostoverride: nil,
				spec:         "fixtures/openstack_api.json",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serv := genServerAPI(tt.fields.spec)
			defer serv.Close()

			u, _ := url.Parse(serv.URL)

			host := u.Host
			if tt.fields.hostoverride != nil {
				host = *tt.fields.hostoverride
			}

			_, err := loadSpec(host)
			if (err != nil) != tt.wantErr {
				t.Errorf("apiLoader.Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
