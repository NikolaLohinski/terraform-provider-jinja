package provider_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/nikolalohinski/terraform-provider-jinja/v2/internal/provider"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var testProviderFactory = map[string]func() (tfprotov6.ProviderServer, error){
	"jinja": providerserver.NewProtocol6WithError(provider.New("test")()),
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
			ProtoV6ProviderFactories: testProviderFactory,
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
	It("should fail with the expected error", func() {
		resource.UnitTest(GinkgoT(), resource.TestCase{
			ProtoV6ProviderFactories: testProviderFactory,
			Steps: []resource.TestStep{
				{
					Config:      *terraformCode,
					ExpectError: regexp.MustCompile(strings.ReplaceAll(errRegex, " ", "\\s")),
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
