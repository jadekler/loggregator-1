package agent_test

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"code.cloudfoundry.org/loggregator/testservers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent Health Endpoint", func() {
	It("returns health metrics", func() {
		consumerServer, err := NewServer()
		Expect(err).ToNot(HaveOccurred())
		defer consumerServer.Stop()
		agentCleanup, agentPorts := testservers.StartAgent(
			testservers.BuildAgentConfig("127.0.0.1", consumerServer.Port()),
		)
		defer agentCleanup()

		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", agentPorts.Health))
		Expect(err).ToNot(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		Expect(body).To(ContainSubstring("dopplerConnections"))
		Expect(body).To(ContainSubstring("dopplerV1Streams"))
		Expect(body).To(ContainSubstring("dopplerV2Streams"))
	})
})
