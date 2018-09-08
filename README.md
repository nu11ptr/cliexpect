# cliexpect 
[![Build Status](https://travis-ci.org/nu11ptr/cliexpect.svg?branch=master)](https://travis-ci.org/nu11ptr/cliexpect) [![Build status](https://ci.appveyor.com/api/projects/status/hcn04efhv6be9qef/branch/master?svg=true)](https://ci.appveyor.com/project/nu11ptr/cliexpect/branch/master) [![Coverage Status](https://coveralls.io/repos/github/nu11ptr/cliexpect/badge.svg?branch=master)](https://coveralls.io/github/nu11ptr/cliexpect?branch=master) [![Maintainability](https://api.codeclimate.com/v1/badges/58fd89136467e9c5f5f2/maintainability)](https://codeclimate.com/github/nu11ptr/cliexpect/maintainability) [![codebeat badge](https://codebeat.co/badges/bc9f0e88-f744-4383-8a81-b0e6672f2fbd)](https://codebeat.co/projects/github-com-nu11ptr-cliexpect-master) [![Go Report Card](https://goreportcard.com/badge/github.com/nu11ptr/cliexpect)](https://goreportcard.com/report/github.com/nu11ptr/cliexpect) [![GoDoc](https://godoc.org/github.com/nu11ptr/cliexpect?status.svg)](https://godoc.org/github.com/nu11ptr/cliexpect)

An expect client designed to work specifically with CLI shell interfaces. Specifically, it always assumes a prompt will separate the data allowing easy traversal of multiple outputs. 

Additionally,
it always matches the text body as a 2nd pass after identifying the placement of it from the prompt. This allows matching to "fail fast" (the alternative is to match the body AND prompt at same time at which point if the body doesn't match even though the prompt does, your program will pause until the timeout is reached waiting to see if new data would match the regex - which it never would).

# Usage

A simple example of a typical use case:

```go
	// Typical CLI output for the 'file' command (minus echo)
	input := `user@host:~$ 
test.py: ASCII text
user@host:~$ `

	sh := cliexpect.New(new(strings.Builder), strings.NewReader(input))
	// Setup the prompt regex based on the expected format
	sh.SetPromptRegex(`\w+@\w+:\S+\$ `)

	// NOTE: In real world, check for errors :-)
	_, groups, _ := sh.Retrieve()
	fmt.Printf("%q\n", groups)

	sh.SendLine("file test.py")

	// Optional - since we now have the exact prompt, we can set it explcitely if we want
	sh.SetPrompt(groups[1])
	// The only thing we know is that it should list 'test.py' in the output - get the rest
	_, groups, _ = sh.ExpectRegex(".*test.py.*")
	fmt.Printf("%q\n", groups)
```

The output is (the first string is matched text, the second the matched prompt):

```go
["" "user@host:~$ "]
["\ntest.py: ASCII text\n" "user@host:~$ "]
```

Play with cliexpect in your browser [here](https://play.jsgo.io/e9846a340391be92f7a311414d36eabe0f7f06d2)

# Status

It is thought to be feature complete and stable and has a comprehensive test suite, however, it has seen very limited real world use. Use at your own risk.
