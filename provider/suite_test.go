package jinja_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	jinja "github.com/nikolalohinski/terraform-provider-jinja/provider"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var testProvider *schema.Provider
var testProviderFactory map[string]func() (*schema.Provider, error)

func init() {
	testProvider = jinja.Provider()
	testProviderFactory = map[string]func() (*schema.Provider, error){
		"jinja": func() (*schema.Provider, error) { return testProvider, nil },
	}
}

func TestJinja(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "jinja")
}

func assertPrettyDiff(expected, got string) bool {
	edits := myers.ComputeEdits("expected", expected, got)
	diffs := gotextdiff.ToUnified("expected", "got", expected, edits)
	return Expect(diffs.Hunks).To(BeEmpty(), "\n"+fmt.Sprint(diffs))
}

func itShouldSetTheExpectedResult(terraformCode *string, expectedResult string) {
	It("should render the expected content", func() {
		resource.UnitTest(GinkgoT(), resource.TestCase{
			ProviderFactories: testProviderFactory,
			Steps: []resource.TestStep{
				{
					Config: *terraformCode,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttrSet("data.jinja_template.test", "id"),
						resource.TestCheckResourceAttrWith("data.jinja_template.test", "result", func(got string) error {
							assertPrettyDiff(expectedResult, got)
							return nil
						}),
					),
				},
			},
		})
	})
}

func itShouldFailToRender(terraformCode *string, errRegex string) {
	It("should render the expected content", func() {
		resource.UnitTest(GinkgoT(), resource.TestCase{
			ProviderFactories: testProviderFactory,
			Steps: []resource.TestStep{
				{
					Config:      *terraformCode,
					ExpectError: regexp.MustCompile(errRegex),
				},
			},
		})
	})
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func MustReturn[I interface{}](i I, err error) I {
	if err != nil {
		panic(err)
	}
	return i
}
