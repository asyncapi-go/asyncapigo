package model

type AsyncApi struct {
	AsyncApi   string             `yaml:"asyncapi,omitempty"`
	Info       Info               `yaml:"info,omitempty"`
	Channels   map[string]Channel `yaml:"channels,omitempty"`
	Components Components         `yaml:"components,omitempty"`
}

type Channel struct {
	Subscribe ChannelAction `yaml:"subscribe,omitempty"`
	Publish   ChannelAction `yaml:"publish,omitempty"`
}

type ChannelAction struct {
	Message Object `yaml:"message,omitempty"`
}

type Info struct {
	Title          string  `yaml:"title,omitempty"`
	Version        string  `yaml:"version,omitempty"`
	Description    string  `yaml:"description,omitempty"`
	TermsOfService string  `yaml:"termsOfService,omitempty"`
	Contact        Contact `yaml:"contact,omitempty"`
	License        License `yaml:"license,omitempty"`
}

type Contact struct {
	Name  string `yaml:"name,omitempty"`
	Url   string `yaml:"url,omitempty"`
	Email string `yaml:"email,omitempty"`
}

type License struct {
	Name string `yaml:"name,omitempty"`
	Url  string `yaml:"url,omitempty"`
}

type Components struct {
	Messages map[string]Message `yaml:"messages,omitempty"`
	Schemas  map[string]Object  `yaml:"schemas,omitempty"`
}

type Message struct {
	Ref string `yaml:"ref,omitempty"`

	Headers     Object `yaml:"headers,omitempty"`
	Tags        []Tag  `yaml:"tags,omitempty"`
	Payload     Object `yaml:"payload,omitempty"`
	Summary     string `yaml:"summary,omitempty"`
	Description string `yaml:"description,omitempty"`
	Name        string `yaml:"name,omitempty"`
	ContentType string `yaml:"contentType,omitempty"`
	Title       string `yaml:"title,omitempty"`
}

type Tag struct {
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type Object struct {
	Ref string `yaml:"$ref,omitempty"`

	OneOf []Object `yaml:"oneOf,omitempty"`
	Enum  []any    `yaml:"enum,omitempty"`

	Description string   `yaml:"description,omitempty"`
	Example     any      `yaml:"example,omitempty"`
	Format      string   `yaml:"format,omitempty"`
	Type        string   `yaml:"type,omitempty"`
	Items       []Object `yaml:"items,omitempty"`
	Required    []string `yaml:"required,omitempty"`
	Maximum     any      `yaml:"maximum,omitempty"`
	Minimum     any      `yaml:"minimum,omitempty"`

	Properties map[string]Object `yaml:"properties,omitempty"`
}
