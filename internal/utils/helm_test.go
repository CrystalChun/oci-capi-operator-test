package utils

import (
	"context"
	"fmt"

	helmclient "github.com/mittwald/go-helm-client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/rest"
)

// Mock Helm Client
type mockHelmClient struct {
	chartExists    bool
	releaseExists  bool
	shouldError    bool
	installedChart *helmclient.ChartSpec
	uninstalled    bool
	addedRepo      *repo.Entry
}

func (m *mockHelmClient) GetChart(name string, options *action.ChartPathOptions) (*chart.Chart, string, error) {
	if m.shouldError {
		return nil, "", fmt.Errorf("mock error")
	}
	if !m.chartExists {
		return nil, "", fmt.Errorf("not found")
	}
	return &chart.Chart{}, "", nil
}

func (m *mockHelmClient) GetRelease(name string) (*release.Release, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	if !m.releaseExists {
		return nil, nil
	}
	return &release.Release{}, nil
}

func (m *mockHelmClient) InstallChart(ctx context.Context, spec *helmclient.ChartSpec, options *helmclient.GenericHelmOptions) (*release.Release, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	m.installedChart = spec
	return &release.Release{}, nil
}

func (m *mockHelmClient) UninstallRelease(spec *helmclient.ChartSpec) error {
	if m.shouldError {
		return fmt.Errorf("mock error")
	}
	m.uninstalled = true
	return nil
}

func (m *mockHelmClient) AddOrUpdateChartRepo(entry repo.Entry) error {
	if m.shouldError {
		return fmt.Errorf("mock error")
	}
	m.addedRepo = &entry
	return nil
}

func (m *mockHelmClient) GetProviders() getter.Providers {
	return getter.Providers{}
}

func (m *mockHelmClient) GetReleaseValues(name string, allValues bool) (map[string]interface{}, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	return map[string]interface{}{}, nil
}

func (m *mockHelmClient) GetSettings() *cli.EnvSettings {
	return cli.New()
}

func (m *mockHelmClient) InstallOrUpgradeChart(ctx context.Context, spec *helmclient.ChartSpec, options *helmclient.GenericHelmOptions) (*release.Release, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	return &release.Release{}, nil
}

func (m *mockHelmClient) LintChart(spec *helmclient.ChartSpec) error {
	if m.shouldError {
		return fmt.Errorf("mock error")
	}
	return nil
}

func (m *mockHelmClient) ListDeployedReleases() ([]*release.Release, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	return []*release.Release{}, nil
}

func (m *mockHelmClient) ListReleaseHistory(name string, max int) ([]*release.Release, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	return []*release.Release{}, nil
}

func (m *mockHelmClient) ListReleasesByStateMask(state action.ListStates) ([]*release.Release, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	return []*release.Release{}, nil
}

func (m *mockHelmClient) RollbackRelease(spec *helmclient.ChartSpec) error {
	if m.shouldError {
		return fmt.Errorf("mock error")
	}
	return nil
}

func (m *mockHelmClient) RunChartTests(releaseName string) (bool, error) {
	if m.shouldError {
		return false, fmt.Errorf("mock error")
	}
	return true, nil
}

func (m *mockHelmClient) SetDebugLog(debugLog action.DebugLog) {
	// No-op for mock
}

func (m *mockHelmClient) TemplateChart(spec *helmclient.ChartSpec, options *helmclient.HelmTemplateOptions) ([]byte, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	return []byte("test-template"), nil
}

func (m *mockHelmClient) UninstallReleaseByName(name string) error {
	if m.shouldError {
		return fmt.Errorf("mock error")
	}
	return nil
}

func (m *mockHelmClient) UpdateChartRepos() error {
	if m.shouldError {
		return fmt.Errorf("mock error")
	}
	return nil
}

func (m *mockHelmClient) UpgradeChart(ctx context.Context, spec *helmclient.ChartSpec, options *helmclient.GenericHelmOptions) (*release.Release, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	return &release.Release{}, nil
}

var _ = Describe("Helm Utils", func() {
	var (
		mockClient *mockHelmClient
	)

	BeforeEach(func() {
		mockClient = &mockHelmClient{}
	})

	Context("GetHelmClient", func() {
		It("should create a helm client with correct namespace", func() {
			cfg := &rest.Config{
				Host: "test-host",
			}
			client, err := GetHelmClient("test-namespace", cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})
	})

	Context("ChartExists", func() {
		It("should return true when chart exists", func() {
			mockClient.chartExists = true
			exists, err := ChartExists(mockClient, "test-chart")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should return false when chart does not exist", func() {
			mockClient.chartExists = false
			exists, err := ChartExists(mockClient, "test-chart")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should return error when client fails", func() {
			mockClient.shouldError = true
			_, err := ChartExists(mockClient, "test-chart")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting chart"))
		})
	})

	Context("AddChartRepo", func() {
		It("should add chart repo successfully", func() {
			err := AddChartRepo(mockClient, "test-repo", "https://test-repo.com")
			Expect(err).NotTo(HaveOccurred())
			Expect(mockClient.addedRepo).NotTo(BeNil())
			Expect(mockClient.addedRepo.Name).To(Equal("test-repo"))
			Expect(mockClient.addedRepo.URL).To(Equal("https://test-repo.com"))
		})

		It("should return error when client fails", func() {
			mockClient.shouldError = true
			err := AddChartRepo(mockClient, "test-repo", "https://test-repo.com")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error adding chart repo"))
		})
	})

	Context("ReleaseExists", func() {
		It("should return true when release exists", func() {
			mockClient.releaseExists = true
			exists, err := ReleaseExists(mockClient, "test-release")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should return false when release does not exist", func() {
			mockClient.releaseExists = false
			exists, err := ReleaseExists(mockClient, "test-release")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should return error when client fails", func() {
			mockClient.shouldError = true
			_, err := ReleaseExists(mockClient, "test-release")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error getting release"))
		})
	})

	Context("InstallHelmChart", func() {
		It("should install chart successfully", func() {
			chartSpec := &helmclient.ChartSpec{
				ReleaseName: "test-release",
				ChartName:   "test-chart",
				Namespace:   "test-namespace",
			}
			err := InstallHelmChart(mockClient, chartSpec)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockClient.installedChart).To(Equal(chartSpec))
		})

		It("should return error when client fails", func() {
			mockClient.shouldError = true
			err := InstallHelmChart(mockClient, &helmclient.ChartSpec{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error installing chart"))
		})
	})

	Context("RemoveHelmChart", func() {
		It("should remove chart successfully", func() {
			chartSpec := &helmclient.ChartSpec{
				ReleaseName: "test-release",
			}
			err := RemoveHelmChart(mockClient, chartSpec)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockClient.uninstalled).To(BeTrue())
		})

		It("should return error when client fails", func() {
			mockClient.shouldError = true
			err := RemoveHelmChart(mockClient, &helmclient.ChartSpec{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error removing chart"))
		})
	})
})
