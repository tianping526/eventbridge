package pattern

import (
	"context"
	"fmt"
	"net"

	"github.com/go-kratos/kratos/v2/log"
)

func init() {
	registerMatchFunc("cidr", newMatchFuncCidr)
}

func newMatchFuncCidr(ctx context.Context, logger *log.Helper, spec interface{}) (matchFunc, error) {
	cidr, ok := spec.(string)
	if !ok {
		return nil, fmt.Errorf("cidr spec(type=%T, val=%v) should be string", spec, spec)
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	return func(val interface{}) (bool, error) {
		ipStr, ok := val.(string)
		if !ok {
			logger.WithContext(ctx).Errorf("ip(type=%T, val=%v) should be string", val, val)
			return false, nil
		}
		ip := net.ParseIP(ipStr)
		if ip == nil {
			logger.WithContext(ctx).Errorf(
				"ipStr(%s) is not a valid textual representation of an IP address", ipStr,
			)
			return false, nil
		}
		return ipNet.Contains(ip), nil
	}, nil
}
