package libauth

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"io"
)

const base64N = "LVoJPiCN2R8G90yg+hmFHuacZ1OWMnrsSTXkYpUq/3dlbfKwv6xztjI7DeBE45QA"

func sha1sum(input string) string {
	h := sha1.New()
	io.WriteString(h, input)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func md5sum(input string) string {
	h := md5.New()
	io.WriteString(h, input)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func QuirkBase64Encode(t string) string {
	a := len(t)
	len := a / 3 * 4
	if a%3 != 0 {
		len += 4
	}
	u := make([]byte, len)
	r := byte('=')
	ui := 0
	for o := 0; o < a; o += 3 {
		var p [3]byte
		p[2] = t[o]
		if o+1 < a {
			p[1] = t[o+1]
		} else {
			p[1] = 0
		}
		if o+2 < a {
			p[0] = t[o+2]
		} else {
			p[0] = 0
		}
		h := int(p[2])<<16 | int(p[1])<<8 | int(p[0])
		for i := 0; i < 4; i++ {
			if o*8+i*6 > a*8 {
				u[ui] = r
			} else {
				u[ui] = base64N[h>>uint(6*(3-i))&0x3F]
			}
			ui++
		}
	}
	return string(u[:len])
}

func XEncode(str, key string) *string {

	S := func(a string, b bool) []uint32 {
		c := len(a)
		v := make([]uint32, (c+3)/4)
		for i := 0; i < c; i += 4 {
			t := uint32(0)
			for j := 0; j+i < c && j < 4; j++ {
				t |= uint32(a[j+i]) << (uint32(j) * 8)
			}
			v[i>>2] = t
		}
		if b {
			v = append(v, uint32(c))
		}
		return v
	}
	L := func(a []uint32, b bool) *string {
		d := len(a)
		c := (d - 1) << 2
		if b {
			m := int(a[d-1])
			if (m < c-3) || (m > c) {
				return nil
			}
			c = m
		}
		var buffer bytes.Buffer
		for i := 0; i < d; i++ {
			buffer.Write([]byte{byte(a[i] & 0xff), byte(a[i] >> 8 & 0xff), byte(a[i] >> 16 & 0xff), byte(a[i] >> 24 & 0xff)})
		}
		var s string
		if b {
			s = buffer.String()[:c]
		} else {
			s = buffer.String()
		}
		return &s
	}

	if len(str) == 0 {
		empty := ""
		return &empty
	}
	v := S(str, true)
	k := S(key, false)
	n := len(v) - 1
	z := v[n]
	y := v[0]
	d := uint32(0)
	for q := 6 + 52/(n+1); q > 0; q-- {
		d += 0x9E3779B9
		e := (d >> 2) & 3
		for p := 0; p <= n; p++ {
			if p == n {
				y = v[0]
			} else {
				y = v[p+1]
			}
			m := (z >> 5) ^ (y << 2)
			m += (y >> 3) ^ (z << 4) ^ (d ^ y)
			m += k[(p&3)^int(e)] ^ z
			v[p] += m
			z = v[p]
		}
	}
	return L(v, false)
}
