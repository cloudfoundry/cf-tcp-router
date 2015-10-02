package syncer_test

import (
	"os"
	"time"

	"github.com/cloudfoundry-incubator/tcp-emitter/syncer"
	"github.com/pivotal-golang/clock/fakeclock"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Syncer", func() {
	var (
		syncerRunner *syncer.Syncer
		syncChannel  chan struct{}
		clock        *fakeclock.FakeClock
		syncInterval time.Duration
		logger       lager.Logger
		process      ifrit.Process
	)

	BeforeEach(func() {
		syncChannel = make(chan struct{})
		clock = fakeclock.NewFakeClock(time.Now())
		syncInterval = 1 * time.Second
		logger = lagertest.NewTestLogger("test")
		syncerRunner = syncer.New(clock, syncInterval, syncChannel, logger)
	})

	AfterEach(func() {
		process.Signal(os.Interrupt)
		Eventually(process.Wait()).Should(Receive(BeNil()))
		close(syncChannel)
	})

	Context("on a specified interval", func() {

		It("should sync", func() {
			process = ifrit.Invoke(syncerRunner)
			var t1 time.Time
			var t2 time.Time

			clock.Increment(syncInterval + 100*time.Millisecond)

			select {
			case <-syncChannel:
				t1 = clock.Now()
			case <-time.After(2 * syncInterval):
				Fail("did not receive a sync event")
			}

			clock.Increment(syncInterval + 100*time.Millisecond)

			select {
			case <-syncChannel:
				t2 = clock.Now()
			case <-time.After(2 * syncInterval):
				Fail("did not receive a sync event")
			}

			Expect(t2.Sub(t1)).To(BeNumerically("~", syncInterval, 100*time.Millisecond))
		})
	})

	Context("on startup", func() {
		var (
			watchChannel chan struct{}
		)
		BeforeEach(func() {
			watchChannel = make(chan struct{})
			go func() {
				select {
				case <-syncChannel:
					watchChannel <- struct{}{}
				}

			}()
		})
		It("should sync", func() {
			process = ifrit.Invoke(syncerRunner)
			Eventually(watchChannel).Should(Receive())
		})
	})
})
