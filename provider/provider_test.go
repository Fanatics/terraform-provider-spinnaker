package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var (
	testAccProviders map[string]terraform.ResourceProvider
	testAccProvider  *schema.Provider
)

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"spinnaker": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := testAccProvider.InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProviderConfigure(t *testing.T) {
	raw := map[string]interface{}{
		"address": "#address",
		"auth": map[string]interface{}{
			"cert_path": os.Getenv("SPINNAKER_CERT"),
			"key_path":  os.Getenv("SPINNAKER_KEY"),
		},
	}
	rawConfig, configErr := config.NewRawConfig(raw)
	if configErr != nil {
		t.Fatal(configErr)
	}
	c := terraform.NewResourceConfig(rawConfig)
	err := testAccProvider.Configure(c)
	if err != nil {
		t.Fatal(err)
	}

	config := testAccProvider.Meta().(*Services).Config
	if config.Address != raw["address"] {
		t.Fatalf("address should be %#v, not %#v", raw["address"], config.Address)
	}

	auth, ok := raw["auth"].(map[string]interface{})
	if !ok {
		t.Fatal("auth is not present")
	}

	if config.Auth.CertPath != auth["cert_path"] {
		t.Fatalf("certPath should be %#v, not %#v", auth["cert_path"], config.Auth.CertPath)
	}
	if config.Auth.KeyPath != auth["key_path"] {
		t.Fatalf("keyPath should be %#v, not %#v", auth["key_path"], config.Auth.KeyPath)
	}
}

func testAccPreCheck(t *testing.T) {
	c := terraform.NewResourceConfig(nil)
	err := testAccProvider.Configure(c)
	if err != nil {
		t.Fatal(err)
	}
}
