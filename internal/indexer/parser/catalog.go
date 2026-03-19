package parser

import "strings"

// CatalogPatterns holds patterns that a catalog entry contributes per language.
type CatalogPatterns struct {
	GoPatterns   []GoCallPattern
	TSPatterns   []TSAPIPattern
	PyPatterns   []PyCallPattern
	JavaPatterns []JavaCallPattern
}

// catalogEntry maps a dependency identifier to its patterns.
type catalogEntry struct {
	matchPrefix bool // if true, match any dep with HasPrefix(dep, key)
	patterns    CatalogPatterns
}

// builtinCatalog maps known OSS library identifiers to detection patterns.
// conn_type is set per pattern — no hardcoded categories.
var builtinCatalog = map[string]catalogEntry{
	// ── Go ──────────────────────────────────────────────────────────────────

	"google.golang.org/grpc": {
		matchPrefix: true,
		patterns: CatalogPatterns{
			GoPatterns: []GoCallPattern{
				{PackageSuffix: "grpc", Functions: []string{"Dial", "DialContext", "NewClient"}, TargetArg: 0, ConnType: "grpc"},
			},
		},
	},

	"github.com/segmentio/kafka-go": {
		patterns: CatalogPatterns{
			GoPatterns: []GoCallPattern{
				{PackageSuffix: "kafka", Functions: []string{"NewReader"}, TargetArg: 0, ConnType: "kafka_consume"},
				{PackageSuffix: "kafka", Functions: []string{"NewWriter"}, TargetArg: 0, ConnType: "kafka_publish"},
			},
		},
	},

	"github.com/IBM/sarama": {
		matchPrefix: true,
		patterns: CatalogPatterns{
			GoPatterns: []GoCallPattern{
				{PackageSuffix: "sarama", Functions: []string{"NewConsumer", "NewConsumerGroup"}, TargetArg: 0, ConnType: "kafka_consume"},
				{PackageSuffix: "sarama", Functions: []string{"NewSyncProducer", "NewAsyncProducer"}, TargetArg: 0, ConnType: "kafka_publish"},
			},
		},
	},

	"github.com/Shopify/sarama": {
		matchPrefix: true,
		patterns: CatalogPatterns{
			GoPatterns: []GoCallPattern{
				{PackageSuffix: "sarama", Functions: []string{"NewConsumer", "NewConsumerGroup"}, TargetArg: 0, ConnType: "kafka_consume"},
				{PackageSuffix: "sarama", Functions: []string{"NewSyncProducer", "NewAsyncProducer"}, TargetArg: 0, ConnType: "kafka_publish"},
			},
		},
	},

	"github.com/confluentinc/confluent-kafka-go": {
		matchPrefix: true,
		patterns: CatalogPatterns{
			GoPatterns: []GoCallPattern{
				{PackageSuffix: "kafka", Functions: []string{"NewConsumer"}, TargetArg: 0, ConnType: "kafka_consume"},
				{PackageSuffix: "kafka", Functions: []string{"NewProducer"}, TargetArg: 0, ConnType: "kafka_publish"},
			},
		},
	},

	"net/http": {
		patterns: CatalogPatterns{
			GoPatterns: []GoCallPattern{
				{PackageSuffix: "http", Functions: []string{"Get", "Post", "Do"}, TargetArg: 0, ConnType: "http_api"},
			},
		},
	},

	"github.com/go-resty/resty": {
		matchPrefix: true,
		patterns: CatalogPatterns{
			GoPatterns: []GoCallPattern{
				{PackageSuffix: "resty", Functions: []string{"New"}, TargetArg: 0, ConnType: "http_api"},
			},
		},
	},

	"github.com/redis/go-redis": {
		matchPrefix: true,
		patterns: CatalogPatterns{
			GoPatterns: []GoCallPattern{
				{PackageSuffix: "redis", Functions: []string{"NewClient", "NewClusterClient"}, TargetArg: 0, ConnType: "redis"},
			},
		},
	},

	"github.com/nats-io/nats.go": {
		matchPrefix: true,
		patterns: CatalogPatterns{
			GoPatterns: []GoCallPattern{
				{PackageSuffix: "nats", Functions: []string{"Connect"}, TargetArg: 0, ConnType: "nats"},
			},
		},
	},

	// ── NPM ─────────────────────────────────────────────────────────────────

	"kafkajs": {
		patterns: CatalogPatterns{
			TSPatterns: []TSAPIPattern{
				{Pattern: `new\s+Kafka\s*\(\s*\{[^}]*brokers\s*:`, ConnType: "kafka_consume", FileGlob: "*.ts"},
				{Pattern: `\.producer\s*\(\s*\)`, ConnType: "kafka_publish", FileGlob: "*.ts"},
				{Pattern: `\.consumer\s*\(\s*\{`, ConnType: "kafka_consume", FileGlob: "*.ts"},
			},
		},
	},

	"@grpc/grpc-js": {
		patterns: CatalogPatterns{
			TSPatterns: []TSAPIPattern{
				{Pattern: "new\\s+\\w+Client\\s*\\(\\s*['\"`]([^'\"`]+)['\"`]", ConnType: "grpc", FileGlob: "*.ts"},
			},
		},
	},

	"axios": {
		patterns: CatalogPatterns{
			TSPatterns: []TSAPIPattern{
				{Pattern: "axios\\.create\\s*\\(\\s*\\{[^}]*baseURL\\s*:\\s*['\"`]([^'\"`]+)['\"`]", ConnType: "http_api", FileGlob: "*.ts"},
			},
		},
	},

	"ioredis": {
		patterns: CatalogPatterns{
			TSPatterns: []TSAPIPattern{
				{Pattern: "new\\s+Redis\\s*\\(\\s*['\"`]?([^'\"`\\s)]+)", ConnType: "redis", FileGlob: "*.ts"},
			},
		},
	},

	"nats": {
		patterns: CatalogPatterns{
			TSPatterns: []TSAPIPattern{
				{Pattern: "connect\\s*\\(\\s*\\{[^}]*servers\\s*:", ConnType: "nats", FileGlob: "*.ts"},
			},
		},
	},

	// ── Python ──────────────────────────────────────────────────────────────

	"grpcio": {
		patterns: CatalogPatterns{
			PyPatterns: []PyCallPattern{
				{ModuleContains: "grpc", CallPattern: "Stub(", TargetArgIndex: 0, ConnType: "grpc"},
				{ModuleContains: "grpc", CallPattern: "insecure_channel(", TargetArgIndex: 0, ConnType: "grpc"},
			},
		},
	},

	"grpcio-tools": {
		patterns: CatalogPatterns{
			PyPatterns: []PyCallPattern{
				{ModuleContains: "grpc", CallPattern: "Stub(", TargetArgIndex: 0, ConnType: "grpc"},
			},
		},
	},

	"kafka-python": {
		patterns: CatalogPatterns{
			PyPatterns: []PyCallPattern{
				{ModuleContains: "kafka", CallPattern: "KafkaConsumer(", TargetArgIndex: 0, ConnType: "kafka_consume"},
				{ModuleContains: "kafka", CallPattern: "KafkaProducer(", TargetArgIndex: 0, ConnType: "kafka_publish"},
			},
		},
	},

	"confluent-kafka": {
		patterns: CatalogPatterns{
			PyPatterns: []PyCallPattern{
				{ModuleContains: "confluent_kafka", CallPattern: "Consumer(", TargetArgIndex: 0, ConnType: "kafka_consume"},
				{ModuleContains: "confluent_kafka", CallPattern: "Producer(", TargetArgIndex: 0, ConnType: "kafka_publish"},
			},
		},
	},

	"httpx": {
		patterns: CatalogPatterns{
			PyPatterns: []PyCallPattern{
				{ModuleContains: "httpx", CallPattern: "Client(", TargetKeyword: "base_url", ConnType: "http_api"},
				{ModuleContains: "httpx", CallPattern: "AsyncClient(", TargetKeyword: "base_url", ConnType: "http_api"},
			},
		},
	},

	"requests": {
		patterns: CatalogPatterns{
			PyPatterns: []PyCallPattern{
				{ModuleContains: "requests", CallPattern: "get(", TargetArgIndex: 0, ConnType: "http_api"},
				{ModuleContains: "requests", CallPattern: "post(", TargetArgIndex: 0, ConnType: "http_api"},
			},
		},
	},

	"aiohttp": {
		patterns: CatalogPatterns{
			PyPatterns: []PyCallPattern{
				{ModuleContains: "aiohttp", CallPattern: "ClientSession(", TargetKeyword: "base_url", ConnType: "http_api"},
			},
		},
	},

	"redis": {
		patterns: CatalogPatterns{
			PyPatterns: []PyCallPattern{
				{ModuleContains: "redis", CallPattern: "Redis(", TargetKeyword: "host", ConnType: "redis"},
				{ModuleContains: "redis", CallPattern: "from_url(", TargetArgIndex: 0, ConnType: "redis"},
			},
		},
	},

	"nats-py": {
		patterns: CatalogPatterns{
			PyPatterns: []PyCallPattern{
				{ModuleContains: "nats", CallPattern: "connect(", TargetArgIndex: 0, ConnType: "nats"},
			},
		},
	},

	// ── Maven / Java ────────────────────────────────────────────────────────

	"io.grpc:grpc": {
		matchPrefix: true,
		patterns: CatalogPatterns{
			JavaPatterns: []JavaCallPattern{
				{ImportContains: "io.grpc", MethodCall: "newBlockingStub", TargetArgIndex: 0, ConnType: "grpc"},
				{ImportContains: "io.grpc", MethodCall: "forAddress", TargetArgIndex: 0, ConnType: "grpc"},
			},
		},
	},

	"org.springframework.kafka": {
		matchPrefix: true,
		patterns: CatalogPatterns{
			JavaPatterns: []JavaCallPattern{
				{ImportContains: "springframework.kafka", Annotation: "KafkaListener", TargetAttribute: "topics", ConnType: "kafka_consume"},
				{ImportContains: "springframework.kafka", MethodCall: "send", TargetArgIndex: 0, ConnType: "kafka_publish"},
			},
		},
	},

	"org.springframework.cloud:spring-cloud-openfeign": {
		matchPrefix: true,
		patterns: CatalogPatterns{
			JavaPatterns: []JavaCallPattern{
				{ImportContains: "openfeign", Annotation: "FeignClient", TargetAttribute: "name", ConnType: "http_api"},
			},
		},
	},

	"org.springframework:spring-web": {
		matchPrefix: true,
		patterns: CatalogPatterns{
			JavaPatterns: []JavaCallPattern{
				{ImportContains: "springframework.web", MethodCall: "getForObject", TargetArgIndex: 0, ConnType: "http_api"},
				{ImportContains: "springframework.web", MethodCall: "postForObject", TargetArgIndex: 0, ConnType: "http_api"},
			},
		},
	},

	"org.apache.kafka:kafka-clients": {
		matchPrefix: true,
		patterns: CatalogPatterns{
			JavaPatterns: []JavaCallPattern{
				{ImportContains: "apache.kafka", MethodCall: "subscribe", TargetArgIndex: 0, ConnType: "kafka_consume"},
				{ImportContains: "apache.kafka", MethodCall: "send", TargetArgIndex: 0, ConnType: "kafka_publish"},
			},
		},
	},

	"io.lettuce:lettuce-core": {
		matchPrefix: true,
		patterns: CatalogPatterns{
			JavaPatterns: []JavaCallPattern{
				{ImportContains: "lettuce", MethodCall: "create", TargetArgIndex: 0, ConnType: "redis"},
			},
		},
	},

	"io.nats:jnats": {
		matchPrefix: true,
		patterns: CatalogPatterns{
			JavaPatterns: []JavaCallPattern{
				{ImportContains: "io.nats", MethodCall: "connect", TargetArgIndex: 0, ConnType: "nats"},
			},
		},
	},
}

// LookupCatalog returns all patterns applicable to the given ProjectDeps.
func LookupCatalog(deps *ProjectDeps) *PatternConfig {
	result := &PatternConfig{}

	for _, dep := range deps.GoModules {
		for key, entry := range builtinCatalog {
			if matchesCatalogKey(dep, key, entry) {
				result.Go = append(result.Go, entry.patterns.GoPatterns...)
			}
		}
	}

	for _, dep := range deps.NPMPkgs {
		for key, entry := range builtinCatalog {
			if dep == key {
				result.TypeScript = append(result.TypeScript, entry.patterns.TSPatterns...)
			}
		}
	}

	for _, dep := range deps.PyPkgs {
		for key, entry := range builtinCatalog {
			if dep == key {
				result.Python = append(result.Python, entry.patterns.PyPatterns...)
			}
		}
	}

	for _, dep := range deps.MavenPkgs {
		for key, entry := range builtinCatalog {
			if matchesCatalogKey(dep, key, entry) {
				result.Java = append(result.Java, entry.patterns.JavaPatterns...)
			}
		}
	}

	return result
}

// matchesCatalogKey returns true if dep matches the catalog key.
func matchesCatalogKey(dep, key string, entry catalogEntry) bool {
	if entry.matchPrefix {
		return strings.HasPrefix(dep, key)
	}
	return dep == key
}
