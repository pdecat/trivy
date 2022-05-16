package report_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	dbTypes "github.com/aquasecurity/trivy-db/pkg/types"
	"github.com/aquasecurity/trivy/pkg/report"
	"github.com/aquasecurity/trivy/pkg/types"
)

func TestReportWriter_Table(t *testing.T) {
	testCases := []struct {
		name               string
		results            types.Results
		expectedOutput     string
		includeNonFailures bool
	}{
		{
			name: "happy path full",
			results: types.Results{
				{
					Target: "test",
					Vulnerabilities: []types.DetectedVulnerability{
						{
							VulnerabilityID:  "CVE-2020-0001",
							PkgName:          "foo",
							InstalledVersion: "1.2.3",
							FixedVersion:     "3.4.5",
							PrimaryURL:       "https://avd.aquasec.com/nvd/cve-2020-0001",
							Vulnerability: dbTypes.Vulnerability{
								Title:       "foobar",
								Description: "baz",
								Severity:    "HIGH",
							},
						},
					},
				},
			},
			expectedOutput: `+---------+------------------+----------+-------------------+---------------+--------------------------------------+
| LIBRARY | VULNERABILITY ID | SEVERITY | INSTALLED VERSION | FIXED VERSION |                TITLE                 |
+---------+------------------+----------+-------------------+---------------+--------------------------------------+
| foo     | CVE-2020-0001    | HIGH     | 1.2.3             | 3.4.5         | foobar                               |
|         |                  |          |                   |               | -->avd.aquasec.com/nvd/cve-2020-0001 |
+---------+------------------+----------+-------------------+---------------+--------------------------------------+
`,
		},
		{
			name: "happy path with filePath in result",
			results: types.Results{
				{
					Target: "test",
					Vulnerabilities: []types.DetectedVulnerability{
						{
							VulnerabilityID:  "CVE-2020-0001",
							PkgName:          "foo",
							PkgPath:          "foo/bar",
							InstalledVersion: "1.2.3",
							FixedVersion:     "3.4.5",
							PrimaryURL:       "https://avd.aquasec.com/nvd/cve-2020-0001",
							Vulnerability: dbTypes.Vulnerability{
								Title:       "foobar",
								Description: "baz",
								Severity:    "HIGH",
							},
						},
					},
				},
			},
			expectedOutput: `+-----------+------------------+----------+-------------------+---------------+--------------------------------------+
|  LIBRARY  | VULNERABILITY ID | SEVERITY | INSTALLED VERSION | FIXED VERSION |                TITLE                 |
+-----------+------------------+----------+-------------------+---------------+--------------------------------------+
| foo (bar) | CVE-2020-0001    | HIGH     | 1.2.3             | 3.4.5         | foobar                               |
|           |                  |          |                   |               | -->avd.aquasec.com/nvd/cve-2020-0001 |
+-----------+------------------+----------+-------------------+---------------+--------------------------------------+
`,
		},
		{
			name: "no title for vuln and missing primary link",
			results: types.Results{
				{
					Target: "test",
					Vulnerabilities: []types.DetectedVulnerability{
						{
							VulnerabilityID:  "CVE-2020-0001",
							PkgName:          "foo",
							InstalledVersion: "1.2.3",
							FixedVersion:     "3.4.5",
							Vulnerability: dbTypes.Vulnerability{
								Description: "foobar",
								Severity:    "HIGH",
							},
						},
					},
				},
			},
			expectedOutput: `+---------+------------------+----------+-------------------+---------------+--------+
| LIBRARY | VULNERABILITY ID | SEVERITY | INSTALLED VERSION | FIXED VERSION | TITLE  |
+---------+------------------+----------+-------------------+---------------+--------+
| foo     | CVE-2020-0001    | HIGH     | 1.2.3             | 3.4.5         | foobar |
+---------+------------------+----------+-------------------+---------------+--------+
`,
		},
		{
			name: "long title for vuln",
			results: types.Results{
				{
					Target: "test",
					Vulnerabilities: []types.DetectedVulnerability{
						{
							VulnerabilityID:  "CVE-2020-1234",
							PkgName:          "foo",
							InstalledVersion: "1.2.3",
							FixedVersion:     "3.4.5",
							PrimaryURL:       "https://avd.aquasec.com/nvd/cve-2020-1234",
							Vulnerability: dbTypes.Vulnerability{
								Title:       "a b c d e f g h i j k l m n o p q r s t u v",
								Description: "foobar",
								Severity:    "HIGH",
							},
						},
					},
				},
			},
			expectedOutput: `+---------+------------------+----------+-------------------+---------------+--------------------------------------+
| LIBRARY | VULNERABILITY ID | SEVERITY | INSTALLED VERSION | FIXED VERSION |                TITLE                 |
+---------+------------------+----------+-------------------+---------------+--------------------------------------+
| foo     | CVE-2020-1234    | HIGH     | 1.2.3             | 3.4.5         | a b c d e f g h i j k l...           |
|         |                  |          |                   |               | -->avd.aquasec.com/nvd/cve-2020-1234 |
+---------+------------------+----------+-------------------+---------------+--------------------------------------+
`,
		},
		{
			name: "happy path misconfigurations",
			results: types.Results{
				{
					Target: "test",
					Misconfigurations: []types.DetectedMisconfiguration{
						{
							Type:       "Kubernetes Security Check",
							ID:         "KSV001",
							Title:      "Image tag ':latest' used",
							Message:    "Message",
							Severity:   "HIGH",
							PrimaryURL: "https://avd.aquasec.com/appshield/ksv001",
							Status:     types.StatusFailure,
						},
						{
							Type:       "Kubernetes Security Check",
							ID:         "KSV002",
							Title:      "SYS_ADMIN capability added",
							Message:    "Message",
							Severity:   "CRITICAL",
							PrimaryURL: "https://avd.aquasec.com/appshield/ksv002",
							Status:     types.StatusFailure,
						},
					},
				},
			},
			expectedOutput: `+---------------------------+------------+----------------------------+----------+------------------------------------------+
|           TYPE            | MISCONF ID |           CHECK            | SEVERITY |                 MESSAGE                  |
+---------------------------+------------+----------------------------+----------+------------------------------------------+
| Kubernetes Security Check |   KSV001   | Image tag ':latest' used   |   HIGH   | Message                                  |
|                           |            |                            |          | -->avd.aquasec.com/appshield/ksv001      |
+                           +------------+----------------------------+----------+------------------------------------------+
|                           |   KSV002   | SYS_ADMIN capability added | CRITICAL | Message                                  |
|                           |            |                            |          | -->avd.aquasec.com/appshield/ksv002      |
+---------------------------+------------+----------------------------+----------+------------------------------------------+
`,
		},
		{
			name:               "happy path misconfigurations with successes",
			includeNonFailures: true,
			results: types.Results{
				{
					Target: "test",
					Misconfigurations: []types.DetectedMisconfiguration{
						{
							Type:       "Kubernetes Security Check",
							ID:         "KSV001",
							Title:      "Image tag ':latest' used",
							Message:    "Message",
							Severity:   "HIGH",
							PrimaryURL: "https://avd.aquasec.com/appshield/ksv001",
							Status:     types.StatusFailure,
						},
						{
							Type:       "Kubernetes Security Check",
							ID:         "KSV002",
							Title:      "SYS_ADMIN capability added",
							Message:    "Message",
							Severity:   "CRITICAL",
							PrimaryURL: "https://avd.aquasec.com/appshield/ksv002",
							Status:     types.StatusPassed,
						},
					},
				},
			},
			expectedOutput: `+---------------------------+------------+----------------------------+----------+--------+------------------------------------------+
|           TYPE            | MISCONF ID |           CHECK            | SEVERITY | STATUS |                 MESSAGE                  |
+---------------------------+------------+----------------------------+----------+--------+------------------------------------------+
| Kubernetes Security Check |   KSV001   | Image tag ':latest' used   |   HIGH   |  FAIL  | Message                                  |
|                           |            |                            |          |        | -->avd.aquasec.com/appshield/ksv001      |
+                           +------------+----------------------------+----------+--------+------------------------------------------+
|                           |   KSV002   | SYS_ADMIN capability added | CRITICAL |  PASS  | Message                                  |
+---------------------------+------------+----------------------------+----------+--------+------------------------------------------+
`,
		},
		{
			name:           "no vulns",
			expectedOutput: ``,
		},
		{
			name: "happy path with vulnerability origin graph",
			results: types.Results{
				{
					Target: "package-lock.json",
					Class:  "lang-pkgs",
					Type:   "npm",
					Vulnerabilities: []types.DetectedVulnerability{
						{
							VulnerabilityID: "CVE-2022-0235",
							PkgID:           "node-fetch@1.7.3",
							PkgName:         "node-fetch",
							Vulnerability: dbTypes.Vulnerability{
								Title:       "foobar",
								Description: "baz",
								Severity:    "HIGH",
							},
							PkgParents: []*types.DependencyTreeItem{
								{
									ID: "isomorphic-fetch@2.2.1",
									Parents: []*types.DependencyTreeItem{
										{
											ID: "fbjs@0.8.18",
											Parents: []*types.DependencyTreeItem{
												{
													ID: "styled-components@3.1.3",
												},
											},
										},
									},
								},
							},
							InstalledVersion: "1.7.3",
							FixedVersion:     "2.6.7, 3.1.1",
						},
						{
							VulnerabilityID: "CVE-2021-26539",
							PkgID:           "sanitize-html@1.20.0",
							PkgName:         "sanitize-html",
							Vulnerability: dbTypes.Vulnerability{
								Title:       "foobar",
								Description: "baz",
								Severity:    "MEDIUM",
							},
							InstalledVersion: "1.20.0",
							FixedVersion:     "2.3.1",
						},
					},
				},
			},
			expectedOutput: `+---------------+------------------+----------+-------------------+---------------+--------+
|    LIBRARY    | VULNERABILITY ID | SEVERITY | INSTALLED VERSION | FIXED VERSION | TITLE  |
+---------------+------------------+----------+-------------------+---------------+--------+
| node-fetch    | CVE-2022-0235    | HIGH     | 1.7.3             | 2.6.7, 3.1.1  | foobar |
+---------------+------------------+----------+-------------------+---------------+        +
| sanitize-html | CVE-2021-26539   | MEDIUM   | 1.20.0            | 2.3.1         |        |
+---------------+------------------+----------+-------------------+---------------+--------+

Vulnerability origin graph:
===========================
package-lock.json
├── node-fetch@1.7.3
│   └── isomorphic-fetch@2.2.1
│       └── fbjs@0.8.18
│           └── styled-components@3.1.3
└── sanitize-html@1.20.0

`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tableWritten := bytes.Buffer{}
			err := report.Write(types.Report{Results: tc.results}, report.Option{
				Format:             "table",
				Output:             &tableWritten,
				IncludeNonFailures: tc.includeNonFailures,
			})
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedOutput, tableWritten.String(), tc.name)
		})
	}
}
