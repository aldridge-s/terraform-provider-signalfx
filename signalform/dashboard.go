package signalform

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

const DASHBOARD_API_URL = "https://api.signalfx.com/v2/dashboard"

func dashboardResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"synced": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the resource in SignalForm and SignalFx are identical or not. Used internally for syncing.",
			},
			"last_updated": &schema.Schema{
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "Latest timestamp the resource was updated",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the dashboard",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the dashboard (Optional)",
			},
			"dashboard_group": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the dashboard group that contains the dashboard. If an ID is not provided during creation, the dashboard will be placed in a newly created dashboard group",
			},
			"charts_resolution": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Specifies the chart data display resolution for charts in this dashboard. Value can be one of \"default\", \"low\", \"high\", or \"highest\". default by default",
				ValidateFunc: validateChartsResolution,
			},
			"time_span_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Type of time interval of the chart. It must be \"absolute\" or \"relative\"",
				ValidateFunc: validateTimeSpanType,
			},
			"time_range": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validateSignalfxRelativeTime,
				Description:   "(time_span_type \"relative\" only) From when to display data. SignalFx time syntax (e.g. -5m, -1h)",
				ConflictsWith: []string{"start_time", "end_time"},
			},
			"start_time": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "(type \"absolute\" only) Seconds since epoch to start the visualization",
				ConflictsWith: []string{"time_range"},
			},
			"end_time": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "(type \"absolute\" only) Seconds since epoch to end the visualization",
				ConflictsWith: []string{"time_range"},
			},
			"chart": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Chart ID and layout information for the charts in the dashboard",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"chart_id": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "ID of the chart to display",
						},
						"row": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "The row to show the chart in (zero-based); if height > 1, this value represents the topmost row of the chart. (greater than or equal to 0)",
						},
						"column": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "The column to show the chart in (zero-based); this value always represents the leftmost column of the chart. (between 0 and 11)",
						},
						"width": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     12,
							Description: "How many columns (out of a total of 12) the chart should take up. (between 1 and 12)",
						},
						"height": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     1,
							Description: "How many rows the chart should take up. (greater than or equal to 1)",
						},
					},
				},
			},
			"variable": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Dashboard variable to apply to each chart in the dashboard",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"property": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "A metric time series dimension or property name",
						},
						"alias": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "An alias for the dashboard variable. This text will appear as the label for the dropdown field on the dashboard",
						},
						"values": &schema.Schema{
							Type:        schema.TypeSet,
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "List of strings (which will be treated as an OR filter on the property)",
						},
						"value_required": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Determines whether a value is required for this variable (and therefore whether it will be possible to view this dashboard without this filter applied). false by default",
						},
						"values_suggested": &schema.Schema{
							Type:        schema.TypeSet,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "A list of strings of suggested values for this variable; these suggestions will receive priority when values are autosuggested for this variable",
						},
						"restricted_suggestions": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "If true, this variable may only be set to the values listed in preferredSuggestions. and only these values will appear in autosuggestion menus. false by default",
						},
					},
				},
			},
			"filter": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Filter to apply to each chart in the dashboard",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"property": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "A metric time series dimension or property name",
						},
						"negated": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "(false by default) Whether this filter should be a \"not\" filter",
						},
						"values": &schema.Schema{
							Type:        schema.TypeSet,
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "List of strings (which will be treated as an OR filter on the property)",
						},
					},
				},
			},
		},

		Create: dashboardCreate,
		Read:   dashboardRead,
		Update: dashboardUpdate,
		Delete: dashboardDelete,
	}
}

/*
  Use Resource object to construct json payload in order to create a dashboard
*/
func getPayloadDashboard(d *schema.ResourceData) ([]byte, error) {
	payload := map[string]interface{}{
		"name":        d.Get("name").(string),
		"description": d.Get("description").(string),
		"groupId":     d.Get("dashboard_group").(string),
	}

	all_filters := make(map[string]interface{})
	if filters := getDashboardFilters(d); len(filters) > 0 {
		all_filters["sources"] = filters
	}
	if variables := getDashboardVariables(d); len(variables) > 0 {
		all_filters["variables"] = variables
	}
	if time := getDashboardTime(d); len(time) > 0 {
		all_filters["time"] = time
	}
	if len(all_filters) > 0 {
		payload["filters"] = all_filters
	}

	if charts := getDashboardCharts(d); len(charts) > 0 {
		payload["charts"] = charts
	}

	if chartsResolution, ok := d.GetOk("charts_resolution"); ok {
		payload["chartDensity"] = strings.ToUpper(chartsResolution.(string))
	}

	return json.Marshal(payload)
}

func getDashboardTime(d *schema.ResourceData) map[string]interface{} {
	timeMap := make(map[string]interface{})
	if val, ok := d.GetOk("time_span_type"); ok {
		if val == "relative" {
			if val, ok := d.GetOk("time_range"); ok {
				timeMap["start"] = val.(string)
				timeMap["end"] = "Now"
			}

		} else {
			if val, ok := d.GetOk("start_time"); ok {
				timeMap["start"] = val.(int) * 1000
			}
			if val, ok := d.GetOk("end_time"); ok {
				timeMap["end"] = val.(int) * 1000
			}
		}
	}

	if len(timeMap) > 0 {
		return timeMap
	}
	return nil
}

func getDashboardCharts(d *schema.ResourceData) []map[string]interface{} {
	charts := d.Get("chart").(*schema.Set).List()
	charts_list := make([]map[string]interface{}, len(charts))
	for i, chart := range charts {
		chart := chart.(map[string]interface{})
		item := make(map[string]interface{})

		item["chartId"] = chart["chart_id"].(string)
		item["row"] = chart["row"].(int)
		item["column"] = chart["column"].(int)
		item["height"] = chart["height"].(int)
		item["width"] = chart["width"].(int)

		charts_list[i] = item
	}
	return charts_list
}

func getDashboardVariables(d *schema.ResourceData) []map[string]interface{} {
	variables := d.Get("variable").(*schema.Set).List()
	vars_list := make([]map[string]interface{}, len(variables))
	for i, variable := range variables {
		variable := variable.(map[string]interface{})
		item := make(map[string]interface{})

		item["property"] = variable["property"].(string)
		item["alias"] = variable["alias"].(string)
		item["value"] = variable["values"].(*schema.Set).List()
		item["required"] = variable["value_required"].(bool)
		if val, ok := variable["values_suggested"]; ok {
			item["preferredSuggestions"] = val.(*schema.Set).List()
		}
		item["restricted"] = variable["restricted_suggestions"].(bool)

		vars_list[i] = item
	}
	return vars_list
}

func getDashboardFilters(d *schema.ResourceData) []map[string]interface{} {
	filters := d.Get("filter").(*schema.Set).List()
	filter_list := make([]map[string]interface{}, len(filters))
	for i, filter := range filters {
		filter := filter.(map[string]interface{})
		item := make(map[string]interface{})

		item["property"] = filter["property"].(string)
		item["NOT"] = filter["negated"].(bool)
		item["value"] = filter["values"].(*schema.Set).List()

		filter_list[i] = item
	}
	return filter_list
}

func dashboardCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalformConfig)
	payload, err := getPayloadDashboard(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	return resourceCreate(DASHBOARD_API_URL, config.SfxToken, payload, d)
}

func dashboardRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalformConfig)
	url := fmt.Sprintf("%s/%s", DASHBOARD_API_URL, d.Id())

	return resourceRead(url, config.SfxToken, d)
}

func dashboardUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalformConfig)
	payload, err := getPayloadDashboard(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}
	url := fmt.Sprintf("%s/%s", DASHBOARD_API_URL, d.Id())

	return resourceUpdate(url, config.SfxToken, payload, d)
}

func dashboardDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalformConfig)
	url := fmt.Sprintf("%s/%s", DASHBOARD_API_URL, d.Id())
	return resourceDelete(url, config.SfxToken, d)
}

/*
  Validate Chart Resolution option against a list of allowed words.
*/
func validateChartsResolution(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	allowedWords := []string{"default", "low", "high", "highest"}
	for _, word := range allowedWords {
		if value == word {
			return
		}
	}
	errors = append(errors, fmt.Errorf("%s not allowed; must be one of: %s", value, strings.Join(allowedWords, ", ")))
	return
}
