package upstream

import (
	"net/url"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	v1_2_0  = "v1.2.0"
	channel = "channel"
)

func Test_parseReplicatedURL(t *testing.T) {
	tests := []struct {
		name                 string
		uri                  string
		expectedAppSlug      string
		expectedChannel      *string
		expectedVersionLabel *string
		expectedSequence     *int
	}{
		{
			name:                 "replicated://app-slug",
			uri:                  "replicated://app-slug",
			expectedAppSlug:      "app-slug",
			expectedChannel:      nil,
			expectedVersionLabel: nil,
			expectedSequence:     nil,
		},
		{
			name:                 "replicated://app-slug@v1.2.0",
			uri:                  "replicated://app-slug@v1.2.0",
			expectedAppSlug:      "app-slug",
			expectedChannel:      nil,
			expectedVersionLabel: &v1_2_0,
			expectedSequence:     nil,
		},
		{
			name:                 "replicated://app-slug/channel",
			uri:                  "replicated://app-slug/channel",
			expectedAppSlug:      "app-slug",
			expectedChannel:      &channel,
			expectedVersionLabel: nil,
			expectedSequence:     nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			u, err := url.ParseRequestURI(test.uri)
			req.NoError(err)

			replicatedUpstream, err := parseReplicatedURL(u)
			req.NoError(err)
			assert.Equal(t, test.expectedAppSlug, replicatedUpstream.AppSlug)

			if test.expectedVersionLabel != nil || replicatedUpstream.VersionLabel != nil {
				assert.Equal(t, test.expectedVersionLabel, replicatedUpstream.VersionLabel)
			}
		})
	}
}

func Test_releaseToFiles(t *testing.T) {
	tests := []struct {
		name     string
		release  *Release
		expected []UpstreamFile
	}{
		{
			name: "with common prefix",
			release: &Release{
				Manifests: map[string][]byte{
					"manifests/deployment.yaml": []byte("---"),
					"manifests/service.yaml":    []byte("---"),
				},
			},
			expected: []UpstreamFile{
				UpstreamFile{
					Path:    "deployment.yaml",
					Content: []byte("---"),
				},
				UpstreamFile{
					Path:    "service.yaml",
					Content: []byte("---"),
				},
			},
		},
		{
			name: "without common prefix",
			release: &Release{
				Manifests: map[string][]byte{
					"manifests/deployment.yaml": []byte("---"),
					"service.yaml":              []byte("---"),
				},
			},
			expected: []UpstreamFile{
				UpstreamFile{
					Path:    "manifests/deployment.yaml",
					Content: []byte("---"),
				},
				UpstreamFile{
					Path:    "service.yaml",
					Content: []byte("---"),
				},
			},
		},
		{
			name: "common prefix, with userdata",
			release: &Release{
				Manifests: map[string][]byte{
					"manifests/deployment.yaml": []byte("---"),
					"manifests/service.yaml":    []byte("---"),
					"userdata/values.yaml":      []byte("---"),
				},
			},
			expected: []UpstreamFile{
				UpstreamFile{
					Path:    "deployment.yaml",
					Content: []byte("---"),
				},
				UpstreamFile{
					Path:    "service.yaml",
					Content: []byte("---"),
				},
				UpstreamFile{
					Path:    "userdata/values.yaml",
					Content: []byte("---"),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actual, err := releaseToFiles(test.release)
			req.NoError(err)

			assert.ElementsMatch(t, test.expected, actual)
		})
	}
}

func Test_createConfigValues(t *testing.T) {
	applicationName := "Test App"

	config := &kotsv1beta1.Config{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "Config",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: applicationName,
		},
		Spec: kotsv1beta1.ConfigSpec{
			Groups: []kotsv1beta1.ConfigGroup{
				kotsv1beta1.ConfigGroup{
					Name:  "group_name",
					Title: "Group Title",
					Items: []kotsv1beta1.ConfigItem{
						// should replace default
						kotsv1beta1.ConfigItem{
							Name:    "1_with_default",
							Type:    "string",
							Default: "default_1_new",
							Value:   "",
						},
						// should preserve value and add default
						kotsv1beta1.ConfigItem{
							Name:    "2_with_value",
							Type:    "string",
							Default: "default_2",
							Value:   "value_2_new",
						},
						// should add a new item
						kotsv1beta1.ConfigItem{
							Name:    "4_with_default",
							Type:    "string",
							Default: "default_4",
						},
					},
				},
			},
		},
	}

	configValues := &kotsv1beta1.ConfigValues{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "ConfigValues",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: applicationName,
		},
		Spec: kotsv1beta1.ConfigValuesSpec{
			Values: map[string]kotsv1beta1.ConfigValue{
				"1_with_default": kotsv1beta1.ConfigValue{
					Default: "default_1",
				},
				"2_with_value": kotsv1beta1.ConfigValue{
					Value: "value_2",
				},
				"3_with_both": kotsv1beta1.ConfigValue{
					Value:   "value_3",
					Default: "default_3",
				},
			},
		},
	}

	req := require.New(t)

	// like new install, should match config
	expected1 := map[string]kotsv1beta1.ConfigValue{
		"1_with_default": kotsv1beta1.ConfigValue{
			Default: "default_1_new",
		},
		"2_with_value": kotsv1beta1.ConfigValue{
			Value:   "value_2_new",
			Default: "default_2",
		},
		"4_with_default": kotsv1beta1.ConfigValue{
			Default: "default_4",
		},
	}
	values1, err := createConfigValues(applicationName, config, nil)
	req.NoError(err)
	assert.Equal(t, expected1, values1.Spec.Values)

	// Like an app without a config, should have exact same values
	expected2 := configValues.Spec.Values
	values2, err := createConfigValues(applicationName, nil, configValues)
	req.NoError(err)
	assert.Equal(t, expected2, values2.Spec.Values)

	// updating existing values with new config, should do a merge
	expected3 := map[string]kotsv1beta1.ConfigValue{
		"1_with_default": kotsv1beta1.ConfigValue{
			Default: "default_1_new",
		},
		"2_with_value": kotsv1beta1.ConfigValue{
			Value:   "value_2",
			Default: "default_2",
		},
		"3_with_both": kotsv1beta1.ConfigValue{
			Value:   "value_3",
			Default: "default_3",
		},
		"4_with_default": kotsv1beta1.ConfigValue{
			Default: "default_4",
		},
	}
	values3, err := createConfigValues(applicationName, config, configValues)
	req.NoError(err)
	assert.Equal(t, expected3, values3.Spec.Values)
}
