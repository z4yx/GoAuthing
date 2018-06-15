package libauth

const base64N = "LVoJPiCN2R8G90yg+hmFHuacZ1OWMnrsSTXkYpUq/3dlbfKwv6xztjI7DeBE45QA"

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
