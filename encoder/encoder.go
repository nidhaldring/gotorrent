package encoder

import (
	"fmt"
	"gotorrent/decoder"
)

func EncodeDict(dict decoder.BencodeDict) string {
	encodedStr := ""
	for k, v := range dict {
		switch val := v.(type) {
		case int:
			encodedStr += fmt.Sprintf("%s%s", encodeString(k), encodeInt(val))
		case string:
			encodedStr += fmt.Sprintf("%s%s", encodeString(k), encodeString(val))
		case []any:
			encodedStr += fmt.Sprintf("%s%s", encodeString(k), encodeList(val))
		case decoder.BencodeDict:
			encodedStr += fmt.Sprintf("%s%s", encodeString(k), EncodeDict(val))
		}
	}

	return "d" + encodedStr + "e"
}

func encodeList(list []any) string {
	encodedStr := ""
	for _, v := range list {
		switch val := v.(type) {
		case int:
			encodedStr += encodeInt(val)
		case string:
			encodedStr += encodeString(val)
		case []any:
			encodedStr += encodeList(val)
		case decoder.BencodeDict:
			encodedStr += EncodeDict(val)
		}
	}

	return "l" + encodedStr + "e"
}

func encodeString(s string) string {
	return fmt.Sprintf("%d:%s", len(s), s)
}

func encodeInt(n int) string {
	return fmt.Sprintf("i%de", n)
}
