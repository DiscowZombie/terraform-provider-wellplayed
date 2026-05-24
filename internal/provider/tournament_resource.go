package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/DiscowZombie/terraform-provider-wellplayed/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the resource satisfies the framework interfaces.
var (
	_ resource.Resource                = &tournamentResource{}
	_ resource.ResourceWithConfigure   = &tournamentResource{}
	_ resource.ResourceWithImportState = &tournamentResource{}
)

// NewTournamentResource constructs the wellplayed_tournament resource.
func NewTournamentResource() resource.Resource {
	return &tournamentResource{}
}

type tournamentResource struct {
	client *client.Client
}

// tournamentModel maps the resource schema to Go values. The framework decodes
// nested blocks into the pointer/slice fields below automatically.
type tournamentModel struct {
	ID                   types.String      `tfsdk:"id"`
	Title                types.String      `tfsdk:"title"`
	Description          types.String      `tfsdk:"description"`
	StartAt              timetypes.RFC3339 `tfsdk:"start_at"`
	EndAt                timetypes.RFC3339 `tfsdk:"end_at"`
	StartRegistrationsAt timetypes.RFC3339 `tfsdk:"start_registrations_at"`
	EndRegistrationsAt   timetypes.RFC3339 `tfsdk:"end_registrations_at"`
	VisibleAt            timetypes.RFC3339 `tfsdk:"visible_at"`
	OrganizationID       types.String      `tfsdk:"organization_id"`
	TournamentSeriesID   types.String      `tfsdk:"tournament_series_id"`
	CreatedByID          types.String      `tfsdk:"created_by_id"`
	CreatedAt            types.String      `tfsdk:"created_at"`
	UpdatedAt            types.String      `tfsdk:"updated_at"`
	Configuration        *configModel      `tfsdk:"configuration"`
}

type configModel struct {
	Type                        types.String                 `tfsdk:"type"`
	TeamMinSize                 types.Float64                `tfsdk:"team_min_size"`
	TeamMaxSize                 types.Float64                `tfsdk:"team_max_size"`
	TeamsCount                  types.Float64                `tfsdk:"teams_count"`
	TeamStatusAfterRegistration types.String                 `tfsdk:"team_status_after_registration"`
	RegistrationConditions      *registrationConditionsModel `tfsdk:"registration_conditions"`
	CustomFields                []propertyModel              `tfsdk:"custom_fields"`
}

type registrationConditionsModel struct {
	TeamConditions   []teamConditionModel   `tfsdk:"team_conditions"`
	MemberConditions []memberConditionModel `tfsdk:"member_conditions"`
}

type teamConditionModel struct {
	Property          types.String               `tfsdk:"property"`
	PropertyCondition types.String               `tfsdk:"property_condition"`
	ErrorMessage      types.String               `tfsdk:"error_message"`
	StringCondition   *stringConditionModel      `tfsdk:"string_condition"`
	NumericCondition  *teamNumericConditionModel `tfsdk:"numeric_condition"`
}

type memberConditionModel struct {
	PropertySource   types.String    `tfsdk:"property_source"`
	PropertySourceID types.String    `tfsdk:"property_source_id"`
	ErrorMessage     types.String    `tfsdk:"error_message"`
	RuleDescription  types.String    `tfsdk:"rule_description"`
	Condition        *conditionModel `tfsdk:"condition"`
}

type conditionModel struct {
	Property          types.String          `tfsdk:"property"`
	PropertyCondition types.String          `tfsdk:"property_condition"`
	StringCondition   *stringConditionModel `tfsdk:"string_condition"`
	NumericCondition  *numberConditionModel `tfsdk:"numeric_condition"`
}

type stringConditionModel struct {
	ConditionType types.String `tfsdk:"condition_type"`
	Value         types.String `tfsdk:"value"`
}

type numberConditionModel struct {
	ConditionType types.String  `tfsdk:"condition_type"`
	Value         types.Float64 `tfsdk:"value"`
}

type teamNumericConditionModel struct {
	AggregationType  types.String  `tfsdk:"aggregation_type"`
	PropertySource   types.String  `tfsdk:"property_source"`
	PropertySourceID types.String  `tfsdk:"property_source_id"`
	ConditionType    types.String  `tfsdk:"condition_type"`
	Value            types.Float64 `tfsdk:"value"`
}

type propertyModel struct {
	Property    types.String  `tfsdk:"property"`
	Name        types.String  `tfsdk:"name"`
	Type        types.String  `tfsdk:"type"`
	Required    types.Bool    `tfsdk:"required"`
	Order       types.Float64 `tfsdk:"order"`
	Unique      types.Bool    `tfsdk:"unique"`
	Visibility  types.String  `tfsdk:"visibility"`
	Editability types.String  `tfsdk:"editability"`
}

func (r *tournamentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tournament"
}

func (r *tournamentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a WellPlayed tournament, including its registration window and full " +
			"configuration (team sizing, registration conditions, and custom fields). Backed by the " +
			"`createTournament`, `updateTournament`, and `deleteTournament` mutations.\n\n" +
			"Timestamps are RFC 3339 and compared by instant, so `2026-06-01T00:00:00Z` and " +
			"`2026-06-01T00:00:00.000Z` are treated as equal.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the tournament.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Display title of the tournament.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the tournament.",
				Required:            true,
			},
			"start_at": schema.StringAttribute{
				MarkdownDescription: "When the tournament starts (RFC 3339).",
				CustomType:          timetypes.RFC3339Type{},
				Optional:            true,
			},
			"end_at": schema.StringAttribute{
				MarkdownDescription: "When the tournament ends (RFC 3339).",
				CustomType:          timetypes.RFC3339Type{},
				Optional:            true,
			},
			"start_registrations_at": schema.StringAttribute{
				MarkdownDescription: "When registrations open (RFC 3339).",
				CustomType:          timetypes.RFC3339Type{},
				Optional:            true,
			},
			"end_registrations_at": schema.StringAttribute{
				MarkdownDescription: "When registrations close (RFC 3339).",
				CustomType:          timetypes.RFC3339Type{},
				Optional:            true,
			},
			"visible_at": schema.StringAttribute{
				MarkdownDescription: "When the tournament becomes visible (RFC 3339). Computed by the API when not set.",
				CustomType:          timetypes.RFC3339Type{},
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "ID of the organization that owns the tournament.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tournament_series_id": schema.StringAttribute{
				MarkdownDescription: "ID of the tournament series this tournament belongs to, if any.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_by_id": schema.StringAttribute{
				MarkdownDescription: "Account ID of the tournament creator.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the tournament was created (RFC 3339).",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the tournament was last updated (RFC 3339).",
				Computed:            true,
			},
			"configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "Tournament configuration: team sizing, registration conditions, and custom fields. " +
					"The API always provisions a default configuration server-side, so this block is managed " +
					"write-only from configuration: it is sent on create/update but not refreshed during Read, " +
					"and out-of-band changes are not detected as drift.",
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "Configuration type. One of `TOURNAMENT`, `STEP`. Defaults to `TOURNAMENT`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("TOURNAMENT"),
					},
					"team_min_size": schema.Float64Attribute{
						MarkdownDescription: "Minimum number of members per team.",
						Optional:            true,
					},
					"team_max_size": schema.Float64Attribute{
						MarkdownDescription: "Maximum number of members per team.",
						Optional:            true,
					},
					"teams_count": schema.Float64Attribute{
						MarkdownDescription: "Maximum number of teams.",
						Optional:            true,
					},
					"team_status_after_registration": schema.StringAttribute{
						MarkdownDescription: "Team status applied after registration. One of `REGISTERED`, " +
							"`AWAITING_FOR_PRESENCE_CONFIRMATION`, `AWAITING_FOR_PAYMENT`.",
						Optional: true,
					},
					"registration_conditions": schema.SingleNestedAttribute{
						MarkdownDescription: "Conditions teams and players must meet to register.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"team_conditions":   teamConditionsSchema(),
							"member_conditions": memberConditionsSchema(),
						},
					},
					"custom_fields": customFieldsSchema(),
				},
			},
		},
	}
}

func teamConditionsSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		MarkdownDescription: "Conditions applied at the team level during registration.",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"property": schema.StringAttribute{
					MarkdownDescription: "Name of the team property to evaluate.",
					Required:            true,
				},
				"property_condition": schema.StringAttribute{
					MarkdownDescription: "Whether the property must exist. One of `EXISTS`, `DONT_EXIST`.",
					Required:            true,
				},
				"error_message": schema.StringAttribute{
					MarkdownDescription: "Custom error message shown when the condition fails.",
					Optional:            true,
				},
				"string_condition":  stringConditionSchema(),
				"numeric_condition": teamNumericConditionSchema(),
			},
		},
	}
}

func memberConditionsSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		MarkdownDescription: "Conditions applied to each team member during registration.",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"property_source": schema.StringAttribute{
					MarkdownDescription: "Source of the player data. One of `PLAYER`, `IDENTITY_PROVIDER`.",
					Required:            true,
				},
				"property_source_id": schema.StringAttribute{
					MarkdownDescription: "Identifier of the data source (e.g. identity provider ID).",
					Optional:            true,
				},
				"error_message": schema.StringAttribute{
					MarkdownDescription: "Custom error message shown when the condition fails.",
					Optional:            true,
				},
				"rule_description": schema.StringAttribute{
					MarkdownDescription: "Human-readable description of the rule.",
					Optional:            true,
				},
				"condition": schema.SingleNestedAttribute{
					MarkdownDescription: "The condition rule evaluated against the player property.",
					Required:            true,
					Attributes: map[string]schema.Attribute{
						"property": schema.StringAttribute{
							MarkdownDescription: "Name of the property to evaluate.",
							Required:            true,
						},
						"property_condition": schema.StringAttribute{
							MarkdownDescription: "Whether the property must exist. One of `EXISTS`, `DONT_EXIST`.",
							Required:            true,
						},
						"string_condition":  stringConditionSchema(),
						"numeric_condition": numberConditionSchema(),
					},
				},
			},
		},
	}
}

func customFieldsSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		MarkdownDescription: "Custom field definitions collected during registration.",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"property": schema.StringAttribute{
					MarkdownDescription: "Unique key identifier for this property.",
					Required:            true,
				},
				"name": schema.StringAttribute{
					MarkdownDescription: "Human-readable display name.",
					Required:            true,
				},
				"type": schema.StringAttribute{
					MarkdownDescription: "Data type. One of `DATE`, `COUNTRY`, `STRING`, `BOOLEAN`, `PHONE`, " +
						"`EMAIL`, `URL`, `NUMBER`.",
					Required: true,
				},
				"required": schema.BoolAttribute{
					MarkdownDescription: "Whether this property must be filled in.",
					Required:            true,
				},
				"order": schema.Float64Attribute{
					MarkdownDescription: "Display order position.",
					Required:            true,
				},
				"unique": schema.BoolAttribute{
					MarkdownDescription: "Whether the value must be unique across all entities.",
					Required:            true,
				},
				"visibility": schema.StringAttribute{
					MarkdownDescription: "Visibility level. One of `PUBLIC`, `OWNER`, `OWNER_OR_PERMISSION`, " +
						"`WITH_PERMISSION`.",
					Optional: true,
				},
				"editability": schema.StringAttribute{
					MarkdownDescription: "Editability rule. One of `ALWAYS`, `ONE_TIME`, `WITH_PERMISSION`.",
					Optional:            true,
				},
			},
		},
	}
}

func stringConditionSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "String comparison applied to the property.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"condition_type": schema.StringAttribute{
				MarkdownDescription: "Comparison operator. One of `EQ`, `NEQ`.",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "String value to compare against.",
				Required:            true,
			},
		},
	}
}

func numberConditionSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Numeric comparison applied to the property.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"condition_type": schema.StringAttribute{
				MarkdownDescription: "Comparison operator. One of `LT`, `BT`, `LTE`, `BTE`, `EQ`, `NEQ`.",
				Required:            true,
			},
			"value": schema.Float64Attribute{
				MarkdownDescription: "Numeric value to compare against.",
				Required:            true,
			},
		},
	}
}

func teamNumericConditionSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Numeric comparison with optional aggregation across team members.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"aggregation_type": schema.StringAttribute{
				MarkdownDescription: "Aggregation method across team members. One of `SUM`, `AVG`, `MIN`, `MAX`.",
				Optional:            true,
			},
			"property_source": schema.StringAttribute{
				MarkdownDescription: "Source of the property data. One of `PLAYER`, `IDENTITY_PROVIDER`.",
				Optional:            true,
			},
			"property_source_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the property data source.",
				Optional:            true,
			},
			"condition_type": schema.StringAttribute{
				MarkdownDescription: "Comparison operator. One of `LT`, `BT`, `LTE`, `BTE`, `EQ`, `NEQ`.",
				Required:            true,
			},
			"value": schema.Float64Attribute{
				MarkdownDescription: "Numeric value to compare against.",
				Required:            true,
			},
		},
	}
}

func (r *tournamentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *tournamentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tournamentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := buildTournamentInput(&plan)

	query := `mutation($input: CreateTournamentInput!) { createTournament(input: $input) { ` + tournamentComputedFields + ` } }`
	var out struct {
		CreateTournament gqlTournament `json:"createTournament"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &out); err != nil {
		resp.Diagnostics.AddError("Unable to create tournament", err.Error())
		return
	}

	applyComputedToModel(&out.CreateTournament, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tournamentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tournamentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: ID!) { tournament(id: $id) { ` + tournamentReadFields + ` } }`
	var out struct {
		Tournament gqlTournament `json:"tournament"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, &out); err != nil {
		if isNotFoundErr(err) {
			tflog.Warn(ctx, "Tournament not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read tournament", err.Error())
		return
	}

	model := modelFromGQL(&out.Tournament)
	// configuration is write-only managed (the API auto-provisions a default
	// config that would otherwise diff against an absent block), so keep
	// whatever the prior state held rather than refreshing it from the API.
	model.Configuration = state.Configuration
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *tournamentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state tournamentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := buildTournamentInput(&plan)

	query := `mutation($id: ID!, $input: UpdateTournamentInput!) { updateTournament(id: $id, input: $input) { ` + tournamentComputedFields + ` } }`
	vars := map[string]any{"id": state.ID.ValueString(), "input": input}
	var out struct {
		UpdateTournament gqlTournament `json:"updateTournament"`
	}
	if err := r.client.Execute(ctx, query, vars, &out); err != nil {
		resp.Diagnostics.AddError("Unable to update tournament", err.Error())
		return
	}

	applyComputedToModel(&out.UpdateTournament, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tournamentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tournamentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: ID!) { deleteTournament(id: $id) }`
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Unable to delete tournament", err.Error())
		return
	}
}

func (r *tournamentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// isNotFoundErr reports whether a GraphQL error indicates the tournament is gone.
func isNotFoundErr(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not found") ||
		strings.Contains(s, "does not exist") ||
		strings.Contains(s, "no tournament")
}
