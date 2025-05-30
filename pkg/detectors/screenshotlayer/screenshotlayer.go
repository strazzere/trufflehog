package screenshotlayer

import (
	"context"
	"fmt"
	regexp "github.com/wasilibs/go-re2"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/trufflesecurity/trufflehog/v3/pkg/common"
	"github.com/trufflesecurity/trufflehog/v3/pkg/detectors"
	"github.com/trufflesecurity/trufflehog/v3/pkg/pb/detectorspb"
)

type Scanner struct{}

// Ensure the Scanner satisfies the interface at compile time.
var _ detectors.Detector = (*Scanner)(nil)

var (
	client = common.SaneHttpClientTimeOut(10 * time.Second)

	// Make sure that your group is surrounded in boundary characters such as below to reduce false positives.
	keyPat = regexp.MustCompile(detectors.PrefixRegex([]string{"screenshotlayer"}) + `\b([a-zA-Z0-9_]{32})\b`)
)

// Keywords are used for efficiently pre-filtering chunks.
// Use identifiers in the secret preferably, or the provider name.
func (s Scanner) Keywords() []string {
	return []string{"screenshotlayer"}
}

// FromData will find and optionally verify ScreenshotLayer secrets in a given set of bytes.
func (s Scanner) FromData(ctx context.Context, verify bool, data []byte) (results []detectors.Result, err error) {
	dataStr := string(data)

	matches := keyPat.FindAllStringSubmatch(dataStr, -1)

	for _, match := range matches {
		resMatch := strings.TrimSpace(match[1])

		s1 := detectors.Result{
			DetectorType: detectorspb.DetectorType_ScreenshotLayer,
			Raw:          []byte(resMatch),
		}

		if verify {
			req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://api.screenshotlayer.com/api/capture?access_key=%s&url=https://google.com", resMatch), nil)
			if err != nil {
				continue
			}
			res, err := client.Do(req)
			if err == nil {
				bodyBytes, err := io.ReadAll(res.Body)
				if err == nil {
					bodyString := string(bodyBytes)
					validResponse := strings.Contains(bodyString, `PNG`)
					defer res.Body.Close()
					if res.StatusCode >= 200 && res.StatusCode < 300 {
						if validResponse {
							s1.Verified = true
						} else {
							s1.Verified = false
						}
					}
				}
			}
		}

		results = append(results, s1)
	}

	return results, nil
}

func (s Scanner) Type() detectorspb.DetectorType {
	return detectorspb.DetectorType_ScreenshotLayer
}

func (s Scanner) Description() string {
	return "ScreenshotLayer is a service that provides website screenshots. Access keys can be used to capture and retrieve screenshots."
}
