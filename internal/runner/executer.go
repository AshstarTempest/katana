package runner

import (
	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/katana/pkg/utils"
	errorutil "github.com/projectdiscovery/utils/errors"
	"github.com/remeh/sizedwaitgroup"
)

// ExecuteCrawling executes the crawling main loop
func (r *Runner) ExecuteCrawling() error {
	if r.options.IncrementalCrawling {
		gologger.Info().Msg("Incremental crawling enabled")
	}

	inputs := r.parseInputs()
	if len(inputs) == 0 {
		return errorutil.New("no input provided for crawling")
	}
	for _, input := range inputs {
		_ = r.state.InFlightUrls.Set(utils.AddSchemeIfNotExists(input), struct{}{})
	}

	defer r.crawler.Close()

	wg := sizedwaitgroup.New(r.options.Parallelism)
	for _, input := range inputs {
		if !r.networkpolicy.Validate(input) {
			gologger.Info().Msgf("Skipping excluded host %s", input)
			continue
		}
		wg.Add()
		input = utils.AddSchemeIfNotExists(input)
		go func(input string) {
			defer wg.Done()

			if err := r.crawler.Crawl(input); err != nil {
				gologger.Warning().Msgf("Could not crawl %s: %s", input, err)
			}
			r.state.InFlightUrls.Delete(input)
			if r.options.IncrementalCrawling {
				_ = r.state.CrawledURLs.Set(input, struct{}{})
			}
		}(input)
	}
	wg.Wait()
	return nil
}
