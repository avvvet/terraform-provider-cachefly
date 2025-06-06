// internal/provider/datasources/origins.go
package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cachefly/cachefly-go-sdk/pkg/cachefly"
	api "github.com/cachefly/cachefly-go-sdk/pkg/cachefly/api/v2_5"

	"github.com/cachefly/terraform-provider-cachefly/internal/provider/models"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &OriginsDataSource{}

func NewOriginsDataSource() datasource.DataSource {
	return &OriginsDataSource{}
}

// OriginsDataSource defines the data source implementation.
type OriginsDataSource struct {
	client *cachefly.Client
}

func (d *OriginsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_origins"
}

func (d *OriginsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "CacheFly Origins data source. List all origin server configurations.",

		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Description: "Filter origins by type (e.g., 'http', 's3', 'gcs').",
				Optional:    true,
			},
			"offset": schema.Int64Attribute{
				Description: "Offset for pagination (default: 0).",
				Optional:    true,
			},
			"limit": schema.Int64Attribute{
				Description: "Limit for pagination (default: API default).",
				Optional:    true,
			},
			"response_type": schema.StringAttribute{
				Description: "Optional response type parameter for the API call.",
				Optional:    true,
			},
			"origins": schema.ListNestedAttribute{
				Description: "List of origins.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the origin.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "Type of origin.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the origin.",
							Computed:    true,
						},
						"host": schema.StringAttribute{
							Description: "Hostname of the origin server.",
							Computed:    true,
						},
						"scheme": schema.StringAttribute{
							Description: "Protocol scheme (http or https).",
							Computed:    true,
						},
						"cache_by_query_param": schema.BoolAttribute{
							Description: "Whether to cache content based on query parameters.",
							Computed:    true,
						},
						"gzip": schema.BoolAttribute{
							Description: "Whether gzip compression is enabled.",
							Computed:    true,
						},
						"ttl": schema.Int64Attribute{
							Description: "Time to live (TTL) in seconds for cached content.",
							Computed:    true,
						},
						"missed_ttl": schema.Int64Attribute{
							Description: "TTL in seconds for missed (404/error) responses.",
							Computed:    true,
						},
						"connection_timeout": schema.Int64Attribute{
							Description: "Connection timeout in seconds.",
							Computed:    true,
						},
						"time_to_first_byte_timeout": schema.Int64Attribute{
							Description: "Time to first byte timeout in seconds.",
							Computed:    true,
						},
						"access_key": schema.StringAttribute{
							Description: "S3 access key (for S3 origins).",
							Computed:    true,
							Sensitive:   true,
						},
						"secret_key": schema.StringAttribute{
							Description: "S3 secret key (for S3 origins).",
							Computed:    true,
							Sensitive:   true,
						},
						"region": schema.StringAttribute{
							Description: "S3 region (for S3 origins).",
							Computed:    true,
						},
						"signature_version": schema.StringAttribute{
							Description: "S3 signature version (for S3 origins).",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "When the origin was created.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "When the origin was last updated.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *OriginsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*cachefly.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *cachefly.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *OriginsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data models.OriginsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading origins data source", map[string]interface{}{
		"type": data.Type.ValueString(),
	})

	// Build options
	opts := api.ListOriginsOptions{
		Type:         data.Type.ValueString(),
		ResponseType: data.ResponseType.ValueString(),
	}

	if !data.Offset.IsNull() {
		opts.Offset = int(data.Offset.ValueInt64())
	}
	if !data.Limit.IsNull() {
		opts.Limit = int(data.Limit.ValueInt64())
	}

	// Get the origins
	originsResp, err := d.client.Origins.List(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading CacheFly Origins",
			"Could not read origins: "+err.Error(),
		)
		return
	}

	// Map response to data model
	origins := make([]attr.Value, len(originsResp.Origins))
	for i, origin := range originsResp.Origins {
		originObj, _ := types.ObjectValue(
			map[string]attr.Type{
				"id":                         types.StringType,
				"type":                       types.StringType,
				"name":                       types.StringType,
				"host":                       types.StringType,
				"scheme":                     types.StringType,
				"cache_by_query_param":       types.BoolType,
				"gzip":                       types.BoolType,
				"ttl":                        types.Int64Type,
				"missed_ttl":                 types.Int64Type,
				"connection_timeout":         types.Int64Type,
				"time_to_first_byte_timeout": types.Int64Type,
				"access_key":                 types.StringType,
				"secret_key":                 types.StringType,
				"region":                     types.StringType,
				"signature_version":          types.StringType,
				"created_at":                 types.StringType,
				"updated_at":                 types.StringType,
			},
			map[string]attr.Value{
				"id":                   types.StringValue(origin.ID),
				"type":                 types.StringValue(origin.Type),
				"name":                 types.StringValue(origin.Name),
				"host":                 types.StringValue(origin.Hostname),
				"scheme":               types.StringValue(origin.Scheme),
				"cache_by_query_param": types.BoolValue(origin.CacheByQueryParam),
				"gzip":                 types.BoolValue(origin.Gzip),
				"ttl":                  types.Int64Value(int64(origin.TTL)),
				"missed_ttl":           types.Int64Value(int64(origin.MissedTTL)),
				"connection_timeout": func() types.Int64 {
					if origin.ConnectionTimeout > 0 {
						return types.Int64Value(int64(origin.ConnectionTimeout))
					}
					return types.Int64Null()
				}(),
				"time_to_first_byte_timeout": func() types.Int64 {
					if origin.TimeToFirstByteTimeout > 0 {
						return types.Int64Value(int64(origin.TimeToFirstByteTimeout))
					}
					return types.Int64Null()
				}(),
				"access_key": func() types.String {
					if origin.AccessKey != "" {
						return types.StringValue(origin.AccessKey)
					}
					return types.StringNull()
				}(),
				"secret_key": func() types.String {
					if origin.SecretKey != "" {
						return types.StringValue(origin.SecretKey)
					}
					return types.StringNull()
				}(),
				"region": func() types.String {
					if origin.Region != "" {
						return types.StringValue(origin.Region)
					}
					return types.StringNull()
				}(),
				"signature_version": func() types.String {
					if origin.SignatureVersion != "" {
						return types.StringValue(origin.SignatureVersion)
					}
					return types.StringNull()
				}(),
				"created_at": types.StringValue(origin.CreatedAt),
				"updated_at": types.StringValue(origin.UpdatedAt),
			},
		)
		origins[i] = originObj
	}

	originsList, diags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":                         types.StringType,
				"type":                       types.StringType,
				"name":                       types.StringType,
				"host":                       types.StringType,
				"scheme":                     types.StringType,
				"cache_by_query_param":       types.BoolType,
				"gzip":                       types.BoolType,
				"ttl":                        types.Int64Type,
				"missed_ttl":                 types.Int64Type,
				"connection_timeout":         types.Int64Type,
				"time_to_first_byte_timeout": types.Int64Type,
				"access_key":                 types.StringType,
				"secret_key":                 types.StringType,
				"region":                     types.StringType,
				"signature_version":          types.StringType,
				"created_at":                 types.StringType,
				"updated_at":                 types.StringType,
			},
		},
		origins,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Origins = originsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
