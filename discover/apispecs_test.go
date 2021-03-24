package discover

import (
	"bytes"
	"net/url"
	"strconv"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/go-openapi/swag"
	"github.com/spf13/viper"

	"github.com/kenjones-cisco/dapperdox/config"
	"github.com/kenjones-cisco/dapperdox/discover/models"
)

func Test_fetchAPISpecs(t *testing.T) {
	_ = config.LoadFixture("../fixtures")
	viper.Set(config.SpecDir, "../tmp/specs")

	type fields struct {
		invalidrewritespath    bool
		invalidrewritesversion bool
	}

	type args struct {
		host       *string
		servicemap *models.ServiceMap
		spec       string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "fail - empty hostname in invalid service map",
			fields: fields{
				invalidrewritespath:    false,
				invalidrewritesversion: false,
			},
			args: args{
				host:       swag.String(""),
				servicemap: nil,
				spec:       "",
			},
			want: 0,
		},
		{
			name: "fail - avoid processing - no responsive API servers",
			fields: fields{
				invalidrewritespath:    false,
				invalidrewritesversion: false,
			},
			args: args{
				host:       swag.String(""),
				servicemap: testServiceMap,
				spec:       "",
			},
			want: 0,
		},
		{
			name: "fail - avoid processing - ignored services",
			fields: fields{
				invalidrewritespath:    false,
				invalidrewritesversion: false,
			},
			args: args{
				host:       swag.String(""),
				servicemap: ignoredServiceMap,
				spec:       "",
			},
			want: 0,
		},
		{
			name: "fail - avoid processing - path to rewrites file is incorrect",
			fields: fields{
				invalidrewritespath:    true,
				invalidrewritesversion: false,
			},
			args: args{
				host:       nil,
				servicemap: nil,
				spec:       "fixtures/petstore_api.json",
			},
			want: 0,
		},
		{
			name: "fail - avoid processing - rewrite spec version not supported",
			fields: fields{
				invalidrewritespath:    false,
				invalidrewritesversion: true,
			},
			args: args{
				host:       nil,
				servicemap: nil,
				spec:       "fixtures/petstore_api.json",
			},
			want: 0,
		},
		{
			name: "success - process petstore API specs",
			fields: fields{
				invalidrewritespath:    false,
				invalidrewritesversion: false,
			},
			args: args{
				host:       nil,
				servicemap: nil,
				spec:       "fixtures/petstore_api.json",
			},
			want: 1,
		},
		{
			name: "success - process iam API specs",
			fields: fields{
				invalidrewritespath:    false,
				invalidrewritesversion: false,
			},
			args: args{
				host:       nil,
				servicemap: nil,
				spec:       "fixtures/iam_api.json",
			},
			want: 1,
		},
		{
			name: "success - process approvals API specs",
			fields: fields{
				invalidrewritespath:    false,
				invalidrewritesversion: false,
			},
			args: args{
				host:       nil,
				servicemap: nil,
				spec:       "fixtures/approvals_api.json",
			},
			want: 1,
		},
		{
			name: "success - process aws-svc API specs",
			fields: fields{
				invalidrewritespath:    false,
				invalidrewritesversion: false,
			},
			args: args{
				host:       nil,
				servicemap: nil,
				spec:       "fixtures/aws_api.json",
			},
			want: 1,
		},
		{
			name: "success - process openstack-svc API specs",
			fields: fields{
				invalidrewritespath:    false,
				invalidrewritesversion: false,
			},
			args: args{
				host:       nil,
				servicemap: nil,
				spec:       "fixtures/openstack_api.json",
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		var svcMap models.ServiceMap

		// if servicemap provided, skip entire API server creation
		if tt.args.servicemap != nil {
			svcMap = *tt.args.servicemap
		} else {
			// create API server
			serv := genServerAPI(tt.args.spec)
			defer serv.Close()

			// fetch host and port details
			u, _ := url.Parse(serv.URL)
			port, _ := strconv.Atoi(u.Port())

			// create service details for API server
			tmpSvc := &models.Service{
				Hostname: u.Hostname(),
				Ports: []*models.Port{
					{
						Name:     "http",
						Port:     port,
						Protocol: models.ProtocolHTTP,
					},
				},
			}

			// set host override value if one is provided
			if tt.args.host != nil {
				tmpSvc.Hostname = *tt.args.host
			}

			// create new servicemap based on service details
			svcMap = models.NewServiceMap(tmpSvc)
		}

		// instantiate a new discoverer instance with servicemap
		d := &Discoverer{
			data: &state{
				services: &svcMap,
			},
			services: &fakeController{},
		}

		t.Run(tt.name, func(t *testing.T) {
			if tt.fields.invalidrewritespath {
				prevRewrites := viper.GetString(config.SpecRewrites)

				viper.Set(config.SpecRewrites, "/fake/path/to/fake/rewrites.yaml")
				defer viper.Set(config.SpecRewrites, prevRewrites)
			}

			if tt.fields.invalidrewritesversion {
				prevRewrites := viper.GetString(config.SpecRewrites)

				viper.Set(config.SpecRewrites, "../fixtures/bad_rewrites.yaml")
				defer viper.Set(config.SpecRewrites, prevRewrites)
			}

			specs := d.fetchAPISpecs()
			if len(specs) != tt.want {
				t.Errorf("discover.fetchAPISpecs() = %v, wantErr %v", len(specs), tt.want)
			}
		})
	}
}

func Test_processSpec(t *testing.T) {
	_ = config.LoadFixture("../fixtures")
	viper.Set(config.SpecDir, "../tmp/specs")

	type args struct {
		hostname    string
		rewriteSpec *spec.Swagger
		svcSpec     *spec.Swagger
	}

	tests := []struct {
		name     string
		args     args
		wantPath string
		want     *spec.Swagger
		wantErr  bool
	}{
		{
			name: "fail - nil swagger spec",
			args: args{
				hostname:    "127.0.0.1",
				rewriteSpec: nil,
				svcSpec:     nil,
			},
			wantPath: "",
			want:     nil,
			wantErr:  true,
		},
		{
			name: "success - hostname - service ip",
			args: args{
				hostname:    "127.0.0.1",
				rewriteSpec: nil,
				svcSpec:     &spec.Swagger{},
			},
			wantPath: "../tmp/specs/127.0.0.1",
			want: &spec.Swagger{
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-groupby": groupByCore,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - hostname - service name",
			args: args{
				hostname:    "myservice",
				rewriteSpec: nil,
				svcSpec:     &spec.Swagger{},
			},
			wantPath: "../tmp/specs/myservice",
			want: &spec.Swagger{
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-groupby": groupByCore,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - remove private API - empty spec",
			args: args{
				hostname:    "127.0.0.1",
				rewriteSpec: nil,
				svcSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Paths: &spec.Paths{
							Paths: map[string]spec.PathItem{
								"/fake/private/api": {
									VendorExtensible: spec.VendorExtensible{
										Extensions: map[string]interface{}{
											"x-visibility": "private",
										},
									},
								},
							},
						},
					},
				},
			},
			wantPath: "../tmp/specs/127.0.0.1",
			want: &spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Paths: &spec.Paths{},
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-groupby": groupByCore,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - remove private API - non-private APIs remain",
			args: args{
				hostname:    "127.0.0.1",
				rewriteSpec: nil,
				svcSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Paths: &spec.Paths{
							Paths: map[string]spec.PathItem{
								"/fake/private/api": {
									VendorExtensible: spec.VendorExtensible{
										Extensions: map[string]interface{}{
											"x-visibility": "private",
										},
									},
								},
								"/second/fake/private/api": {
									VendorExtensible: spec.VendorExtensible{
										Extensions: map[string]interface{}{
											"x-visibility": "private",
										},
									},
								},
								"/fake/public/api": {
									VendorExtensible: spec.VendorExtensible{
										Extensions: map[string]interface{}{
											"x-visibility": "public",
										},
									},
								},
								"/fake/api/no/extensions": {},
							},
						},
					},
				},
			},
			wantPath: "../tmp/specs/127.0.0.1",
			want: &spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Paths: &spec.Paths{
						Paths: map[string]spec.PathItem{
							"/fake/public/api": {
								VendorExtensible: spec.VendorExtensible{
									Extensions: map[string]interface{}{
										"x-visibility": "public",
									},
								},
							},
							"/fake/api/no/extensions": {},
						},
					},
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-groupby": groupByCore,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - remove private method - path with no methods",
			args: args{
				hostname:    "127.0.0.1",
				rewriteSpec: nil,
				svcSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Paths: &spec.Paths{
							Paths: map[string]spec.PathItem{
								"/fake/api": {
									PathItemProps: spec.PathItemProps{
										Get: &spec.Operation{
											VendorExtensible: spec.VendorExtensible{
												Extensions: map[string]interface{}{
													"x-visibility": "private",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantPath: "../tmp/specs/127.0.0.1",
			want: &spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Paths: &spec.Paths{
						Paths: map[string]spec.PathItem{
							"/fake/api": {},
						},
					},
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-groupby": groupByCore,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - remove private methods - non-private methods remain",
			args: args{
				hostname:    "127.0.0.1",
				rewriteSpec: nil,
				svcSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Paths: &spec.Paths{
							Paths: map[string]spec.PathItem{
								"/fake/api": {
									PathItemProps: spec.PathItemProps{
										Get: &spec.Operation{
											VendorExtensible: spec.VendorExtensible{
												Extensions: map[string]interface{}{
													"x-visibility": "private",
												},
											},
										},
										Post: &spec.Operation{
											VendorExtensible: spec.VendorExtensible{
												Extensions: map[string]interface{}{
													"x-visibility": "public",
												},
											},
										},
										Patch: &spec.Operation{},
									},
								},
							},
						},
					},
				},
			},
			wantPath: "../tmp/specs/127.0.0.1",
			want: &spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Paths: &spec.Paths{
						Paths: map[string]spec.PathItem{
							"/fake/api": {
								PathItemProps: spec.PathItemProps{
									Post: &spec.Operation{
										VendorExtensible: spec.VendorExtensible{
											Extensions: map[string]interface{}{
												"x-visibility": "public",
											},
										},
									},
									Patch: &spec.Operation{},
								},
							},
						},
					},
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-groupby": groupByCore,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - no private methods - all methods remain",
			args: args{
				hostname:    "127.0.0.1",
				rewriteSpec: nil,
				svcSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Paths: &spec.Paths{
							Paths: map[string]spec.PathItem{
								"/fake/api": {
									PathItemProps: spec.PathItemProps{
										Get:    &spec.Operation{},
										Post:   &spec.Operation{},
										Patch:  &spec.Operation{},
										Put:    &spec.Operation{},
										Delete: &spec.Operation{},
									},
								},
							},
						},
					},
				},
			},
			wantPath: "../tmp/specs/127.0.0.1",
			want: &spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Paths: &spec.Paths{
						Paths: map[string]spec.PathItem{
							"/fake/api": {
								PathItemProps: spec.PathItemProps{
									Get:    &spec.Operation{},
									Post:   &spec.Operation{},
									Patch:  &spec.Operation{},
									Put:    &spec.Operation{},
									Delete: &spec.Operation{},
								},
							},
						},
					},
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-groupby": groupByCore,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - remove private methods - path with no methods",
			args: args{
				hostname:    "127.0.0.1",
				rewriteSpec: nil,
				svcSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Paths: &spec.Paths{
							Paths: map[string]spec.PathItem{
								"/fake/api": {
									PathItemProps: spec.PathItemProps{
										Get: &spec.Operation{
											VendorExtensible: spec.VendorExtensible{
												Extensions: map[string]interface{}{
													"x-visibility": "private",
												},
											},
										},
										Post: &spec.Operation{
											VendorExtensible: spec.VendorExtensible{
												Extensions: map[string]interface{}{
													"x-visibility": "private",
												},
											},
										},
										Patch: &spec.Operation{
											VendorExtensible: spec.VendorExtensible{
												Extensions: map[string]interface{}{
													"x-visibility": "private",
												},
											},
										},
										Put: &spec.Operation{
											VendorExtensible: spec.VendorExtensible{
												Extensions: map[string]interface{}{
													"x-visibility": "private",
												},
											},
										},
										Delete: &spec.Operation{
											VendorExtensible: spec.VendorExtensible{
												Extensions: map[string]interface{}{
													"x-visibility": "private",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantPath: "../tmp/specs/127.0.0.1",
			want: &spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Paths: &spec.Paths{
						Paths: map[string]spec.PathItem{
							"/fake/api": {
								PathItemProps: spec.PathItemProps{},
							},
						},
					},
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-groupby": groupByCore,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - remove private definition - empty spec",
			args: args{
				hostname:    "127.0.0.1",
				rewriteSpec: nil,
				svcSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Definitions: spec.Definitions{
							"object1": spec.Schema{
								VendorExtensible: spec.VendorExtensible{
									Extensions: map[string]interface{}{
										"x-visibility": "private",
									},
								},
							},
						},
					},
				},
			},
			wantPath: "../tmp/specs/127.0.0.1",
			want: &spec.Swagger{
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-groupby": groupByCore,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - remove private definition - non-private definitions remain",
			args: args{
				hostname:    "127.0.0.1",
				rewriteSpec: nil,
				svcSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Definitions: spec.Definitions{
							"object1": spec.Schema{
								VendorExtensible: spec.VendorExtensible{
									Extensions: map[string]interface{}{
										"x-visibility": "private",
									},
								},
							},
							"object2": spec.Schema{
								VendorExtensible: spec.VendorExtensible{
									Extensions: map[string]interface{}{
										"x-visibility": "public",
									},
								},
							},
							"object3": spec.Schema{},
						},
					},
				},
			},
			wantPath: "../tmp/specs/127.0.0.1",
			want: &spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Definitions: spec.Definitions{
						"object2": spec.Schema{
							VendorExtensible: spec.VendorExtensible{
								Extensions: map[string]interface{}{
									"x-visibility": "public",
								},
							},
						},
						"object3": spec.Schema{},
					},
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-groupby": groupByCore,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - set grouping extension - core",
			args: args{
				hostname:    "myservice",
				rewriteSpec: nil,
				svcSpec: &spec.Swagger{
					VendorExtensible: spec.VendorExtensible{
						Extensions: map[string]interface{}{
							"x-mcmp-component-type": "core",
						},
					},
				},
			},
			wantPath: "../tmp/specs/myservice",
			want: &spec.Swagger{
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-mcmp-component-type": "core",
						"x-groupby":             groupByCore,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - set grouping extension - private cloud",
			args: args{
				hostname:    "myservice",
				rewriteSpec: nil,
				svcSpec: &spec.Swagger{
					VendorExtensible: spec.VendorExtensible{
						Extensions: map[string]interface{}{
							"x-mcmp-component-type": "private-cloud",
						},
					},
				},
			},
			wantPath: "../tmp/specs/myservice",
			want: &spec.Swagger{
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-mcmp-component-type": "private-cloud",
						"x-groupby":             groupByPrivate,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - set grouping extension - public cloud",
			args: args{
				hostname:    "myservice",
				rewriteSpec: nil,
				svcSpec: &spec.Swagger{
					VendorExtensible: spec.VendorExtensible{
						Extensions: map[string]interface{}{
							"x-mcmp-component-type": "public-cloud",
						},
					},
				},
			},
			wantPath: "../tmp/specs/myservice",
			want: &spec.Swagger{
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-mcmp-component-type": "public-cloud",
						"x-groupby":             groupByPublic,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - set grouping extension - approvals custom grouping",
			args: args{
				hostname:    "myservice",
				rewriteSpec: nil,
				svcSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Tags: []spec.Tag{
							{
								TagProps: spec.TagProps{
									Name: "approvals",
								},
							},
						},
					},
					VendorExtensible: spec.VendorExtensible{
						Extensions: map[string]interface{}{
							"x-mcmp-component-type": "core",
						},
					},
				},
			},
			wantPath: "../tmp/specs/myservice",
			want: &spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Tags: []spec.Tag{
						{
							TagProps: spec.TagProps{
								Name: "approvals",
							},
						},
					},
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-mcmp-component-type": "core",
						"x-groupby":             "Service Provider APIs",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - set grouping extension - reporting custom grouping",
			args: args{
				hostname:    "myservice",
				rewriteSpec: nil,
				svcSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Tags: []spec.Tag{
							{
								TagProps: spec.TagProps{
									Name: "reporting",
								},
							},
						},
					},
					VendorExtensible: spec.VendorExtensible{
						Extensions: map[string]interface{}{
							"x-mcmp-component-type": "core",
						},
					},
				},
			},
			wantPath: "../tmp/specs/myservice",
			want: &spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Tags: []spec.Tag{
						{
							TagProps: spec.TagProps{
								Name: "reporting",
							},
						},
					},
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-mcmp-component-type": "core",
						"x-groupby":             "Service Provider APIs",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - set grouping extension - fake custom grouping",
			args: args{
				hostname:    "myservice",
				rewriteSpec: nil,
				svcSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Tags: []spec.Tag{
							{
								TagProps: spec.TagProps{
									Name: "fakeserviceone",
								},
							},
						},
					},
					VendorExtensible: spec.VendorExtensible{
						Extensions: map[string]interface{}{
							"x-mcmp-component-type": "private-cloud",
						},
					},
				},
			},
			wantPath: "../tmp/specs/myservice",
			want: &spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Tags: []spec.Tag{
						{
							TagProps: spec.TagProps{
								Name: "fakeserviceone",
							},
						},
					},
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-mcmp-component-type": "private-cloud",
						"x-groupby":             "Custom Group 1",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - apply rewrites - replacing schemes",
			args: args{
				hostname: "127.0.0.1",
				rewriteSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Schemes: []string{"http", "ftp"},
					},
				},
				svcSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Schemes: []string{"https"},
						Security: []map[string][]string{
							{
								"private-security": []string{
									"should be removed",
								},
							},
						},
						SecurityDefinitions: spec.SecurityDefinitions{
							"to-be-replaced-token": &spec.SecurityScheme{
								SecuritySchemeProps: spec.SecuritySchemeProps{},
							},
						},
					},
				},
			},
			wantPath: "../tmp/specs/127.0.0.1",
			want: &spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Schemes: []string{"http", "ftp"},
					Security: []map[string][]string{
						{
							"private-security": []string{
								"should be removed",
							},
						},
					},
					SecurityDefinitions: spec.SecurityDefinitions{
						"to-be-replaced-token": &spec.SecurityScheme{
							SecuritySchemeProps: spec.SecuritySchemeProps{},
						},
					},
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-groupby": groupByCore,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - apply rewrites - replacing security",
			args: args{
				hostname: "127.0.0.1",
				rewriteSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Security: []map[string][]string{
							{
								"token": nil,
							},
						},
					},
				},
				svcSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Schemes: []string{"https"},
						Security: []map[string][]string{
							{
								"private-security": []string{
									"should be removed",
								},
							},
						},
						SecurityDefinitions: spec.SecurityDefinitions{
							"to-be-replaced-token": &spec.SecurityScheme{
								SecuritySchemeProps: spec.SecuritySchemeProps{},
							},
						},
					},
				},
			},
			wantPath: "../tmp/specs/127.0.0.1",
			want: &spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Schemes: []string{"https"},
					Security: []map[string][]string{
						{
							"token": nil,
						},
					},
					SecurityDefinitions: spec.SecurityDefinitions{
						"to-be-replaced-token": &spec.SecurityScheme{
							SecuritySchemeProps: spec.SecuritySchemeProps{},
						},
					},
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-groupby": groupByCore,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "success - apply rewrites - replacing security definitions",
			args: args{
				hostname: "127.0.0.1",
				rewriteSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						SecurityDefinitions: spec.SecurityDefinitions{
							"token": &spec.SecurityScheme{
								SecuritySchemeProps: spec.SecuritySchemeProps{
									Type:     "oath2",
									TokenURL: "https://fakessourl.fake.com/as/token.oauth2",
									Flow:     "password",
									Scopes: map[string]string{
										"tenant:admin":          "Tenant Administrator",
										"tenant:operator":       "Tenant Operator",
										"tenant:view":           "Tenant View",
										"servicegroup:admin":    "Service Group Admin",
										"servicegroup:operator": "Service Group Operator",
										"servicegroup:view":     "Service Group View",
										"funds:admin":           "Funds Admin",
									},
								},
							},
						},
					},
				},
				svcSpec: &spec.Swagger{
					SwaggerProps: spec.SwaggerProps{
						Schemes: []string{"https"},
						Security: []map[string][]string{
							{
								"private-security": []string{
									"should be removed",
								},
							},
						},
						SecurityDefinitions: spec.SecurityDefinitions{
							"to-be-replaced-token": &spec.SecurityScheme{
								SecuritySchemeProps: spec.SecuritySchemeProps{},
							},
						},
					},
				},
			},
			wantPath: "../tmp/specs/127.0.0.1",
			want: &spec.Swagger{
				SwaggerProps: spec.SwaggerProps{
					Schemes: []string{"https"},
					Security: []map[string][]string{
						{
							"private-security": []string{
								"should be removed",
							},
						},
					},
					SecurityDefinitions: spec.SecurityDefinitions{
						"token": &spec.SecurityScheme{
							SecuritySchemeProps: spec.SecuritySchemeProps{
								Type:     "oath2",
								TokenURL: "https://fakessourl.fake.com/as/token.oauth2",
								Flow:     "password",
								Scopes: map[string]string{
									"tenant:admin":          "Tenant Administrator",
									"tenant:operator":       "Tenant Operator",
									"tenant:view":           "Tenant View",
									"servicegroup:admin":    "Service Group Admin",
									"servicegroup:operator": "Service Group Operator",
									"servicegroup:view":     "Service Group View",
									"funds:admin":           "Funds Admin",
								},
							},
						},
					},
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: map[string]interface{}{
						"x-groupby": groupByCore,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, data, err := processSpec(tt.args.hostname, tt.args.rewriteSpec, tt.args.svcSpec)
			if (err != nil) != tt.wantErr {
				t.Errorf("processSpec() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if path != tt.wantPath {
				t.Errorf("processSpec() got path = %v, want path %v", path, tt.wantPath)

				return
			}

			if tt.want != nil {
				wantData, _ := tt.want.MarshalJSON()

				if res := bytes.Compare(data, wantData); res != 0 {
					t.Errorf("processSpec() got = %v, want %v", string(data), string(wantData))
				}
			}
		})
	}
}
