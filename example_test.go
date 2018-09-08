package cliexpect_test

import (
	"fmt"
	"strings"

	"github.com/nu11ptr/cliexpect"
)

func ExampleShell() {
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
	// Output:
	// ["" "user@host:~$ "]
	// ["\ntest.py: ASCII text\n" "user@host:~$ "]
}
