package handler_test

import (
	"github.com/go-chassis/go-chassis/control"
	_ "github.com/go-chassis/go-chassis/control/archaius"
	"github.com/go-chassis/go-chassis/core/config"
	"github.com/go-chassis/go-chassis/core/config/model"
	"github.com/go-chassis/go-chassis/core/handler"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/core/lager"
	"github.com/go-chassis/go-chassis/examples/schemas/helloworld"
	_ "github.com/go-chassis/go-chassis/initiator"
	"github.com/go-chassis/go-chassis/pkg/circuit"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"testing"
)

func TestCBInit(t *testing.T) {
	f := prepareConfDir(t)
	microContent := `---
service_description:
  name: Client
  version: 0.1`
	circuitContent :=
		`
cse:
  isolation:
    Consumer:
      timeoutInMilliseconds: 1000
      maxConcurrentRequests: 100
      Server:
        timeoutInMilliseconds: 1000
        maxConcurrentRequests: 100
  circuitBreaker:
    Consumer:
      enabled: false
      forceOpen: false
      forceClosed: false
      sleepWindowInMilliseconds: 10000
      requestVolumeThreshold: 20
      errorThresholdPercentage: 10
      Server:
        enabled: true
        forceOpen: false
        forceClosed: false
        sleepWindowInMilliseconds: 10000
        requestVolumeThreshold: 20
        errorThresholdPercentage: 50
  #容错处理函数，目前暂时按照开源的方式来不进行区分处理，统一调用fallback函数
  fallback:
    Consumer:
      enabled: true
  fallbackpolicy:
    Consumer:
      policy: throwexception
`
	prepareTestFile(t, f, "chassis.yaml", "")
	prepareTestFile(t, f, "microservice.yaml", microContent)
	prepareTestFile(t, f, "circuit_breaker.yaml", circuitContent)
	err := config.Init()
	assert.NoError(t, err)
	circuit.Init()
	opts := control.Options{
		Infra:   config.GlobalDefinition.Panel.Infra,
		Address: config.GlobalDefinition.Panel.Settings["address"],
	}
	err = control.Init(opts)
	assert.NoError(t, err)
}

func TestBizKeeperConsumerHandler_Handle(t *testing.T) {
	t.Log("testing bizkeeper consumer handler")
	c := handler.Chain{}
	c.AddHandler(&handler.BizKeeperConsumerHandler{})

	config.GlobalDefinition = &model.GlobalCfg{}
	config.GlobalDefinition.Cse.Handler.Chain.Consumer = make(map[string]string)
	config.GlobalDefinition.Cse.Handler.Chain.Consumer["bizkeeperconsumerdefault"] = "bizkeeper-consumer"
	i := &invocation.Invocation{
		MicroServiceName: "service1",
		SchemaID:         "schema1",
		OperationID:      "SayHello",
		Args:             &helloworld.HelloRequest{Name: "peter"},
	}

	c.Next(i, func(r *invocation.Response) error {
		assert.NoError(t, r.Err)
		log.Println(r.Result)
		return r.Err
	})
}
func TestBizKeeperProviderHandler_Handle(t *testing.T) {
	t.Log("testing bizkeeper provider handler")

	c := handler.Chain{}
	c.AddHandler(&handler.BizKeeperProviderHandler{})

	config.GlobalDefinition = &model.GlobalCfg{}
	config.GlobalDefinition.Cse.Handler.Chain.Provider = make(map[string]string)
	config.GlobalDefinition.Cse.Handler.Chain.Provider["bizkeeperproviderdefault"] = "bizkeeper-provider"
	i := &invocation.Invocation{
		MicroServiceName: "service1",
		SchemaID:         "schema1",
		OperationID:      "SayHello",
		Args:             &helloworld.HelloRequest{Name: "peter"},
	}

	c.Next(i, func(r *invocation.Response) error {
		assert.NoError(t, r.Err)
		log.Println(r.Result)
		return r.Err
	})
}

func TestBizKeeperHandler_Names(t *testing.T) {
	bizPro := &handler.BizKeeperProviderHandler{}
	proName := bizPro.Name()
	assert.Equal(t, "bizkeeper-provider", proName)

	bizCon := &handler.BizKeeperConsumerHandler{}
	conName := bizCon.Name()
	assert.Equal(t, "bizkeeper-consumer", conName)

}
func init() {
	lager.Init(&lager.Options{
		LoggerLevel:   "INFO",
		RollingPolicy: "size",
	})
}
func BenchmarkBizKeepConsumerHandler_Handler(b *testing.B) {
	b.Log("benchmark for bizkeeper consumer handler")
	c := handler.Chain{}
	c.AddHandler(&handler.BizKeeperConsumerHandler{})
	gopath := os.Getenv("GOPATH")
	os.Setenv("CHASSIS_HOME", gopath+"/src/github.com/go-chassis/go-chassis/examples/discovery/client/")

	config.Init()
	opts := control.Options{
		Infra:   config.GlobalDefinition.Panel.Infra,
		Address: config.GlobalDefinition.Panel.Settings["address"],
	}
	control.Init(opts)
	inv := &invocation.Invocation{
		MicroServiceName: "fakeService",
		SchemaID:         "schema",
		OperationID:      "SayHello",
		Args:             &helloworld.HelloRequest{Name: "peter"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Next(inv, func(r *invocation.Response) error {
			return r.Err
		})
		inv.HandlerIndex = 0
	}
}
