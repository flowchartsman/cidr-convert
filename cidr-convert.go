// This file is in the public domain.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
)

type node struct {
	sub [2]*node
}

var (
	none = &node{}
	all  = &node{}
)

type Tree struct {
	root *node
}

func NewTree() *Tree {
	return &Tree{none}
}

func (t *Tree) insert_addr(a uint) {
	add_to_node(&t.root, a, 31, -1)
}

func (t *Tree) insert_range(start uint, end uint) {

	var (
		bit int32
		m   uint
		tt  uint
	)

	for start <= end {
		m = (start - 1) & ^start
		for start+m > end {
			m >>= 1
		}
		bit = -1
		tt = m
		for tt != 0 {
			bit++
			tt >>= 1
		}
		add_to_node(&t.root, start, 31, bit)
		start += m + 1
	}
}

func (t *Tree) insert_CIDR(a uint, n int) {
	if n != 0 {
		a &= 0xffffffff & (0xffffffff << (32 - uint(n)))
	} else {
		a = 0
	}
	add_to_node(&t.root, a, 31, int32(31-n))
}

func (t *Tree) DumpTree() {
	dump_tree(t.root, 0, 31)
}

func add_to_node(np **node, a uint, bit int32, end int32) {
	var n *node
	n = *np

	if n == all {
		return
	}
	if bit <= end {
		*np = all
		return
	}
	if n == none {
		n = &node{[2]*node{none, none}}
		*np = n
	}
	add_to_node(&n.sub[(a>>uint(bit))&1], a, bit-1, end)
	if n.sub[0] == all && n.sub[1] == all {
		*np = all
	}
}

func (t *Tree) FindNode(a uint) bool {
	n := t.root
	switch {
	case n == all:
		return true
	case n == none:
		return false
	}
	for bit := 31; bit >= 0; bit-- { //is >= correct?
		//some case where this is the exact node
		n = n.sub[(a>>uint(bit))&1]
		switch {
		case n == all:
			return true
		case n == none:
			return false
		}
	}
	return false
}

func dump_tree(n *node, v uint, bit uint) {
	switch {
	case n == none:
		return
	case n == all:
		fmt.Printf("%d.%d.%d.%d/%d\n", v>>24&0xff, (v>>16)&0xff, (v>>8)&0xff, v&0xff, 31-bit)
		return
	case bit < 0:
		panic("non-leaf node found at the bottom of the tree")
	}
	dump_tree(n.sub[0], v, bit-1)
	dump_tree(n.sub[1], v|(1<<bit), bit-1)
}

func byte4ToInt(b []byte) uint {
	return uint(b[0])<<24 | uint(b[1])<<16 | uint(b[2])<<8 | uint(b[3])
}

func ipstrToInt(ipStr string) (ipInt uint, err error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		err = errors.New("Invalid IP address")
		return
	}
	ipInt = byte4ToInt(ip.To4())
	return
}

func main() {
	t := NewTree()
	stdin := bufio.NewScanner(os.Stdin)
	// TODO: technically '192.168.1.0-192.168.1.255/8' will be accepted as valid
	// (it will just ignore the CIDR block). Make this an invalid line
	extractor := regexp.MustCompile(`^(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})(?:-(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}))?(?:/(\d{1,2}))?$`)
	for stdin.Scan() {
		matches := extractor.FindStringSubmatch(stdin.Text())
		//fmt.Printf("%#v\n", matches[1:])
		switch {
		case matches[1] == "":
			// Invalid or empty line
			continue
		case matches[2] != "":
			// IP-IP range
			ip1i, err := ipstrToInt(matches[1])
			ip2i, err := ipstrToInt(matches[2])
			if err != nil || ip1i >= ip2i {
				continue
			}
			t.insert_range(ip1i, ip2i)
		case matches[3] != "":
			// IP/CIDR
			ipi, err := ipstrToInt(matches[1])
			if err != nil {
				continue
			}
			ci, err := strconv.ParseUint(matches[3], 10, 32)
			if err != nil {
				continue
			}
			t.insert_CIDR(ipi, int(ci))
		default:
			// single IP
			ipi, err := ipstrToInt(matches[1])
			if err != nil {
				continue
			}
			t.insert_addr(ipi)
		}
	}

	t.DumpTree()
}
