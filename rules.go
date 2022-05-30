package a2s

import (
	"encoding/binary"
	"errors"
)

const (
	A2S_RULES_REQUEST  = 0x56
	A2S_RULES_RESPONSE = 0x45
)

var (
	ErrBadRulesReply = errors.New("Bad rules reply")
)

type RulesInfo struct {
	// Number of rules in the response.
	Count uint16 `json:"Count"`

	// KV map of rules name to value
	Rules map[string]string `json:"Rules"`
}

type Rule struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

func (c *Client) QueryRules() (*RulesInfo, error) {
	return queryRules(c, []byte{0xFF, 0xFF, 0xFF, 0xFF})
}

func queryRules(c *Client, code []byte) (*RulesInfo, error) {
	/*
		A2S_RULES

		Request Format

		Header	byte	'V' (0x56)
		Challenge	int	Challenge number, or -1 (0xFFFFFFFF) to receive a challenge number.

		FF FF FF FF 56 FF FF FF FF                         ÿÿÿÿVÿÿÿÿ"

		Example A2S_PLAYER request with the received challenge number:

		FF FF FF FF 56 4B A1 D5 22                         ÿÿÿÿVK¡Õ"
	*/

	ruleRequest := []byte{0xFF, 0xFF, 0xFF, 0xFF, A2S_RULES_REQUEST}
	ruleRequest = append(ruleRequest, code...)

	data, immediate, err := c.getChallenge(ruleRequest, A2S_RULES_RESPONSE)

	if err != nil {
		return nil, err
	}

	if !immediate {
		return queryRules(c, data)
	}

	// Read header (long 4 bytes)
	switch int32(binary.LittleEndian.Uint32(data)) {
	case -1:
		return parseRulesInfo(data)
	case -2:
		data, err = c.collectMultiplePacketResponse(data)

		if err != nil {
			return nil, err
		}

		return parseRulesInfo(data)
	}

	return nil, ErrBadPacketHeader
}

func parseRulesInfo(data []byte) (*RulesInfo, error) {
	reader := NewPacketReader(data)

	// Simple response now

	if reader.ReadInt32() != -1 {
		return nil, ErrBadPacketHeader
	}

	if reader.ReadUint8() != A2S_RULES_RESPONSE {
		return nil, ErrBadRulesReply
	}

	rules := &RulesInfo{}

	rules.Count = reader.ReadUint16()

	rules.Rules = make(map[string]string, rules.Count)

	for i := 0; i < int(rules.Count); i++ {
		key, ok := reader.TryReadString()

		if !ok {
			break
		}

		val, ok := reader.TryReadString()

		if !ok {
			break
		}

		rules.Rules[key] = val
	}

	return rules, nil
}
