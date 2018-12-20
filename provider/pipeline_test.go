package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPipeline_basic(t *testing.T) {
	app := "app"
	name := fmt.Sprintf("tf-acc-test-%s",
		acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	newName := name + "-changed"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPipelineDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPipelineConfigBasic(app, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPipelineExists("spinnaker_pipeline.test"),
					resource.TestCheckResourceAttr("spinnaker_pipeline.test", "name", name),
					resource.TestCheckResourceAttr("spinnaker_pipeline.test", "application", app),
				),
			},
			{
				Config: testAccPipelineConfigBasic(app, newName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPipelineExists("spinnaker_pipeline.test"),
					resource.TestCheckResourceAttr("spinnaker_pipeline.test", "name", newName),
					resource.TestCheckResourceAttr("spinnaker_pipeline.test", "application", app),
				),
			},
		},
	})
}

func TestAccPipeline_trigger(t *testing.T) {
	app := "app"
	name := fmt.Sprintf("tf-acc-test-%s",
		acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	newName := name + "-changed"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPipelineDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPipelineConfigTrigger(app, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPipelineExists("spinnaker_pipeline.test"),
					resource.TestCheckResourceAttr("spinnaker_pipeline.test", "name", name),
					resource.TestCheckResourceAttr("spinnaker_pipeline.test", "application", app),
					testAccCheckPipelineTriggers("spinnaker_pipeline.test", []string{
						"spinnaker_pipeline_trigger.jenkins",
					}),
				),
			},
			{
				Config: testAccPipelineConfigTrigger(app, newName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPipelineExists("spinnaker_pipeline.test"),
					resource.TestCheckResourceAttr("spinnaker_pipeline.test", "name", newName),
					resource.TestCheckResourceAttr("spinnaker_pipeline.test", "application", app),
					testAccCheckPipelineTriggers("spinnaker_pipeline.test", []string{
						"spinnaker_pipeline_trigger.jenkins",
					}),
				),
			},
		},
	})
}

func testAccPipelineConfigBasic(app string, name string) string {
	return fmt.Sprintf(`
resource "spinnaker_pipeline" "test" {
	application = "%s"
	name        = "%s"
	index       = 2
}
`, app, name)
}

func testAccPipelineConfigTrigger(app string, name string) string {
	return fmt.Sprintf(`
resource "spinnaker_pipeline" "test" {
	application = "%s"
	name        = "%s"
	index       = 2
}

resource "spinnaker_pipeline_trigger" "jenkins" {
	pipeline = "${spinnaker_pipeline.test.id}"
	job = "Bridge Career/job/Bridge_nav/job/Bridge_nav_postmerge"
	master = "inst-ci"
	type = "jenkins"
}
`, app, name)
}

func testAccCheckPipelineExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		pipelineService := testAccProvider.Meta().(*Services).PipelineService
		_, err := pipelineService.GetPipeline(rs.Primary.Attributes["application"], rs.Primary.Attributes["name"])
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckPipelineDestroy(s *terraform.State) error {
	pipelineService := testAccProvider.Meta().(*Services).PipelineService
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "spinnaker_pipeline" {
			_, err := pipelineService.GetPipeline(rs.Primary.Attributes["application"], rs.Primary.Attributes["name"])
			if err == nil {
				return fmt.Errorf("Pipeline still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}