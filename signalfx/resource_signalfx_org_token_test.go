package signalfx

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	sfx "github.com/signalfx/signalfx-go"
)

const newOrgTokenConfig = `
resource "signalfx_org_token" "myorgtokenTOK1" {
  name = "FarToken"
  description = "Farts"
	notifications = ["Email,foo-alerts@example.com"]

  host_or_usage_limits {
    host_limit = 100
    host_notification_threshold = 90
    container_limit = 200
    container_notification_threshold = 180
    custom_metrics_limit = 1000
    custom_metrics_notification_threshold = 900
    high_res_metrics_limit = 1000
    high_res_metrics_notification_threshold = 900
  }
}
`

const updatedOrgTokenConfig = `
resource "signalfx_org_token" "myorgtokenTOK1" {
  name = "FarToken NEW"
  description = "Farts"
	notifications = ["Email,foo-alerts@example.com"]

  host_or_usage_limits {
    host_limit = 100
    host_notification_threshold = 90
    container_limit = 200
    container_notification_threshold = 180
    custom_metrics_limit = 1000
    custom_metrics_notification_threshold = 900
    high_res_metrics_limit = 1000
    high_res_metrics_notification_threshold = 900
  }
}
`

func TestAccCreateUpdateOrgToken(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOrgTokenDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newOrgTokenConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.myorgtokenTOK1", "name", "FarToken"),
					resource.TestCheckResourceAttr("signalfx_org_token.myorgtokenTOK1", "description", "Farts"),
				),
			},
			{
				ResourceName:      "signalfx_org_token.myorgtokenTOK1",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_org_token.myorgtokenTOK1"),
				ImportStateVerify: true,
			},
			// Update Everything
			{
				Config: updatedOrgTokenConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.myorgtokenTOK1", "name", "FarToken NEW"),
					resource.TestCheckResourceAttr("signalfx_org_token.myorgtokenTOK1", "description", "Farts"),
				),
			},
		},
	})
}

func testAccCheckOrgTokenResourceExists(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_org_token":
			tok, err := client.GetOrgToken(rs.Primary.ID)
			if tok.Name != rs.Primary.ID || err != nil {
				return fmt.Errorf("Error finding org token %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	return nil
}

func testAccOrgTokenDestroy(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_org_token":
			tok, _ := client.GetOrgToken(rs.Primary.ID)
			if tok != nil {
				return fmt.Errorf("Found deleted org token %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
