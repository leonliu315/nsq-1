package main

import (
	"../nsq"
	"../util"
	"github.com/bmizerany/assert"
	"io/ioutil"
	"log"
	"net"
	"os"
	"testing"
)

// exercise the basic operations of the V2 protocol
func TestBasicV2(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Equal(t, err, nil)
	tcpAddr := tcpListener.Addr().(*net.TCPAddr)
	defer tcpListener.Close()

	go util.TcpServer(tcpListener, tcpClientHandler)

	msg := nsq.NewMessage(util.Uuid(), []byte("test body"))
	topic := GetTopic("test_v2", 10, os.TempDir())
	topic.PutMessage(msg)

	consumer := nsq.NewConsumer(tcpAddr)

	err = consumer.Connect()
	assert.Equal(t, err, nil)

	err = consumer.Version(nsq.ProtocolV2Magic)
	assert.Equal(t, err, nil)

	err = consumer.WriteCommand(consumer.Subscribe("test_v2", "ch"))
	assert.Equal(t, err, nil)

	err = consumer.WriteCommand(consumer.Ready(1))
	assert.Equal(t, err, nil)

	resp, err := consumer.ReadResponse()
	assert.Equal(t, err, nil)
	frameType, msgInterface, err := consumer.UnpackResponse(resp)
	msgOut := msgInterface.(*nsq.Message)
	assert.Equal(t, frameType, nsq.FrameTypeMessage)
	assert.Equal(t, msgOut.Uuid, msg.Uuid)
	assert.Equal(t, msgOut.Body, msg.Body)
	assert.Equal(t, msgOut.Retries, uint16(1))
}

func TestMultipleConsumerV2(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	msgChan := make(chan *nsq.Message)

	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Equal(t, err, nil)
	tcpAddr := tcpListener.Addr().(*net.TCPAddr)
	defer tcpListener.Close()

	go util.TcpServer(tcpListener, tcpClientHandler)

	msg := nsq.NewMessage(util.Uuid(), []byte("test body"))
	topic := GetTopic("test_multiple_v2", 10, os.TempDir())
	topic.GetChannel("ch1", 10, os.TempDir())
	topic.GetChannel("ch2", 10, os.TempDir())
	topic.PutMessage(msg)

	for _, i := range []string{"1", "2"} {
		consumer := nsq.NewConsumer(tcpAddr)
		err = consumer.Connect()
		assert.Equal(t, err, nil)

		err = consumer.Version(nsq.ProtocolV2Magic)
		assert.Equal(t, err, nil)

		err = consumer.WriteCommand(consumer.Subscribe("test_multiple_v2", "ch"+i))
		assert.Equal(t, err, nil)

		err = consumer.WriteCommand(consumer.Ready(1))
		assert.Equal(t, err, nil)

		go func(c *nsq.Consumer) {
			resp, _ := c.ReadResponse()
			_, msgInterface, _ := c.UnpackResponse(resp)
			msgChan <- msgInterface.(*nsq.Message)
		}(consumer)
	}

	msgOut := <-msgChan
	assert.Equal(t, msgOut.Uuid, msg.Uuid)
	assert.Equal(t, msgOut.Body, msg.Body)
	assert.Equal(t, msgOut.Retries, uint16(1))
	msgOut = <-msgChan
	assert.Equal(t, msgOut.Uuid, msg.Uuid)
	assert.Equal(t, msgOut.Body, msg.Body)
	assert.Equal(t, msgOut.Retries, uint16(1))
}