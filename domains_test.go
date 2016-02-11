package main

import (
	"testing"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
	"time"
	"io/ioutil"
	"crypto/rand"
	"os"
)

const updateFile = "shared/pages/.update"

func TestReadProjects(t *testing.T) {
	*pagesDomain = "test.io"

	d := make(domains)
	err := d.ReadGroups()
	require.NoError(t, err)

	var domains []string
	for domain, _ := range d {
		domains = append(domains, domain)
	}

	expectedDomains := []string{
		"group.test.io",
		"group.internal.test.io",
		"test.domain.com", // from config.json
		"other.domain.com",
	}

	for _, expected := range domains {
		assert.Contains(t, domains, expected)
	}

	for _, actual := range domains {
		assert.Contains(t, expectedDomains, actual)
	}
}

func writeRandomTimestamp() {
	b := make([]byte, 10)
	rand.Read(b)
	ioutil.WriteFile(updateFile, b, 0600)
}

func TestWatchDomains(t *testing.T) {
	update := make(chan domains)
	go watchDomains(func(domains domains) {
		update <- domains
	}, time.Microsecond)

	defer os.Remove(updateFile)

	domains := <-update
	assert.NotNil(t, domains, "if the domains are fetched on start")

	writeRandomTimestamp()
	domains = <-update
	assert.NotNil(t, domains, "if the domains are updated after the creation")

	writeRandomTimestamp()
	domains = <-update
	assert.NotNil(t, domains, "if the domains are updated after the timestamp change")

	os.Remove(updateFile)
	domains = <-update
	assert.NotNil(t, domains, "if the domains are updated after the timestamp removal")
}
