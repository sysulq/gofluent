package parsers

import (
	"github.com/moriyoshi/ik"
	"github.com/pbnjay/strptime"
	"regexp"
	"errors"
	"log"
	"time"
)

type RegexpLineParserPlugin struct {}

type RegexpLineParserFactory struct {
	plugin     *RegexpLineParserPlugin
	logger     *log.Logger
	timeParser func (value string) (time.Time, error)
	regex      *regexp.Regexp
}

type RegexpLineParser struct {
	factory  *RegexpLineParserFactory
	receiver func (ik.FluentRecord) error
}

func (parser *RegexpLineParser) Feed(line string) error {
	regex := parser.factory.regex
	g := regex.FindStringSubmatch(line)
	data := make(map[string]interface {})
	if g == nil {
		parser.factory.logger.Println("Unparsed line: " + line)
		return nil
	}
	for i, name := range regex.SubexpNames() {
		data[name] = g[i]
	}
	parser.receiver(ik.FluentRecord {
		Tag: "",
		Timestamp: 0,
		Data: data,
	})
	return nil
}

func (*RegexpLineParserPlugin) Name() string {
	return "regexp"
}

func (factory *RegexpLineParserFactory) New(receiver func (ik.FluentRecord) error) (ik.LineParser, error) {
	return &RegexpLineParser {
		factory: factory,
		receiver: receiver,
	}, nil
}

func (plugin *RegexpLineParserPlugin) OnRegistering(visitor func (name string, factoryFactory ik.LineParserFactoryFactory) error) error {
	return visitor("regexp", func (engine ik.Engine, config *ik.ConfigElement) (ik.LineParserFactory, error) {
		return plugin.New(engine, config)
	})
}

func (plugin *RegexpLineParserPlugin) newRegexpLineParserFactory(logger *log.Logger, timeParser func (value string) (time.Time, error), regex *regexp.Regexp) (*RegexpLineParserFactory, error) {
	return &RegexpLineParserFactory {
		plugin:     plugin,
		logger:     logger,
		timeParser: timeParser,
		regex:      regex,
	}, nil
}

func (plugin *RegexpLineParserPlugin) New(engine ik.Engine, config *ik.ConfigElement) (ik.LineParserFactory, error) {
	timeFormatStr, ok := config.Attrs["time_format"]
	var timeParser func (value string) (time.Time, error)
	if ok {
		timeParser = func (value string) (time.Time, error) {
			return strptime.Parse(value, timeFormatStr)
		}
	} else {
		timeParser = func (value string) (time.Time, error) {
			// FIXME
			return time.Parse(time.RFC3339, value)
		}
	}
	regexStr, ok := config.Attrs["regexp"]
	if !ok {
		return nil, errors.New("Required attribute `regexp' not found")
	}
	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, err
	}
	return plugin.newRegexpLineParserFactory(engine.Logger(), timeParser, regex)
}

var _ = AddPlugin(&RegexpLineParserPlugin{})
