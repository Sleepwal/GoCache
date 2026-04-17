package server

import (
	"fmt"
	"strconv"
	"strings"

	"GoCache/cache"
	"GoCache/resp"
)

func (ts *TCPServer) cmdSetBit(c *client, args []string) error {
	if len(args) < 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'setbit' command")
	}
	offset, err := strconv.Atoi(args[1])
	if err != nil {
		return c.writer.WriteError("ERR bit offset is not an integer or out of range")
	}
	value, err := strconv.Atoi(args[2])
	if err != nil || (value != 0 && value != 1) {
		return c.writer.WriteError("ERR bit is not an integer or out of range")
	}
	oldBit, err := ts.bitmapCache.SetBit(args[0], offset, value)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	return c.writer.WriteInteger(int64(oldBit))
}

func (ts *TCPServer) cmdGetBit(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'getbit' command")
	}
	offset, err := strconv.Atoi(args[1])
	if err != nil {
		return c.writer.WriteError("ERR bit offset is not an integer or out of range")
	}
	bit, err := ts.bitmapCache.GetBit(args[0], offset)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	return c.writer.WriteInteger(int64(bit))
}

func (ts *TCPServer) cmdBitCount(c *client, args []string) error {
	if len(args) < 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'bitcount' command")
	}
	if len(args) >= 3 {
		start, err1 := strconv.Atoi(args[1])
		end, err2 := strconv.Atoi(args[2])
		if err1 != nil || err2 != nil {
			return c.writer.WriteError("ERR value is not an integer or out of range")
		}
		count, err := ts.bitmapCache.BitCount(args[0], start, end)
		if err != nil {
			return c.writer.WriteError(err.Error())
		}
		return c.writer.WriteInteger(int64(count))
	}
	count, err := ts.bitmapCache.BitCountAll(args[0])
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	return c.writer.WriteInteger(int64(count))
}

func (ts *TCPServer) cmdBitOp(c *client, args []string) error {
	if len(args) < 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'bitop' command")
	}
	result, err := ts.bitmapCache.BitOp(args[0], args[1], args[2:]...)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	return c.writer.WriteInteger(int64(result))
}

func (ts *TCPServer) cmdBitPos(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'bitpos' command")
	}
	bit, err := strconv.Atoi(args[1])
	if err != nil || (bit != 0 && bit != 1) {
		return c.writer.WriteError("ERR bit argument is not an integer or out of range")
	}
	start := 0
	end := -1
	endGiven := false
	if len(args) >= 3 {
		start, err = strconv.Atoi(args[2])
		if err != nil {
			return c.writer.WriteError("ERR value is not an integer or out of range")
		}
	}
	if len(args) >= 4 {
		end, err = strconv.Atoi(args[3])
		if err != nil {
			return c.writer.WriteError("ERR value is not an integer or out of range")
		}
		endGiven = true
	}
	pos, err := ts.bitmapCache.BitPos(args[0], bit, start, end, endGiven)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	return c.writer.WriteInteger(int64(pos))
}

func (ts *TCPServer) cmdPFAdd(c *client, args []string) error {
	if len(args) < 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'pfadd' command")
	}
	result, err := ts.hllCache.PFAdd(args[0], args[1:]...)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	return c.writer.WriteInteger(int64(result))
}

func (ts *TCPServer) cmdPFCount(c *client, args []string) error {
	if len(args) < 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'pfcount' command")
	}
	count, err := ts.hllCache.PFCount(args...)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	return c.writer.WriteInteger(count)
}

func (ts *TCPServer) cmdPFMerge(c *client, args []string) error {
	if len(args) < 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'pfmerge' command")
	}
	_, err := ts.hllCache.PFMerge(args[0], args[1:]...)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	return c.writer.WriteSimpleString("OK")
}

func (ts *TCPServer) cmdGeoAdd(c *client, args []string) error {
	if len(args) < 4 || (len(args)-1)%3 != 0 {
		return c.writer.WriteError("ERR wrong number of arguments for 'geoadd' command")
	}
	var members []cache.GeoMember
	for i := 1; i < len(args); i += 3 {
		lng, err1 := strconv.ParseFloat(args[i], 64)
		lat, err2 := strconv.ParseFloat(args[i+1], 64)
		if err1 != nil || err2 != nil {
			return c.writer.WriteError("ERR invalid longitude/latitude value")
		}
		members = append(members, cache.GeoMember{Name: args[i+2], Longitude: lng, Latitude: lat})
	}
	added, err := ts.geoCache.GeoAdd(args[0], members...)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	return c.writer.WriteInteger(int64(added))
}

func (ts *TCPServer) cmdGeoDist(c *client, args []string) error {
	if len(args) < 3 {
		return c.writer.WriteError("ERR wrong number of arguments for 'geodist' command")
	}
	unit := cache.UnitM
	if len(args) >= 4 {
		unit = cache.GeoUnit(strings.ToLower(args[3]))
	}
	dist, err := ts.geoCache.GeoDist(args[0], args[1], args[2], unit)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	if dist == 0 {
		return c.writer.WriteBulkString("")
	}
	return c.writer.WriteBulkString(fmt.Sprintf("%.4f", dist))
}

func (ts *TCPServer) cmdGeoHash(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'geohash' command")
	}
	hashes, err := ts.geoCache.GeoHash(args[0], args[1:]...)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	c.writer.StartArray(len(hashes))
	for _, h := range hashes {
		c.writer.WriteBulkString(h)
	}
	return nil
}

func (ts *TCPServer) cmdGeoPos(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'geopos' command")
	}
	positions, err := ts.geoCache.GeoPos(args[0], args[1:]...)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	c.writer.StartArray(len(positions))
	for _, pos := range positions {
		if pos == nil {
			c.writer.WriteBulkString("")
		} else {
			c.writer.StartArray(2)
			c.writer.WriteBulkString(fmt.Sprintf("%.6f", pos.Longitude))
			c.writer.WriteBulkString(fmt.Sprintf("%.6f", pos.Latitude))
		}
	}
	return nil
}

func (ts *TCPServer) cmdGeoRadius(c *client, args []string) error {
	if len(args) < 5 {
		return c.writer.WriteError("ERR wrong number of arguments for 'georadius' command")
	}
	key := args[0]
	lng, err1 := strconv.ParseFloat(args[1], 64)
	lat, err2 := strconv.ParseFloat(args[2], 64)
	radius, err3 := strconv.ParseFloat(args[3], 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return c.writer.WriteError("ERR invalid float value")
	}
	unit := cache.GeoUnit(strings.ToLower(args[4]))
	withCoord, withDist, count := parseGeoRadiusOpts(args[5:])
	results, err := ts.geoCache.GeoRadius(key, lng, lat, radius, unit, withCoord, withDist, count)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	return writeGeoResults(c, results, withDist, withCoord)
}

func (ts *TCPServer) cmdGeoRadiusByMember(c *client, args []string) error {
	if len(args) < 4 {
		return c.writer.WriteError("ERR wrong number of arguments for 'georadiusbymember' command")
	}
	radius, err1 := strconv.ParseFloat(args[2], 64)
	unit := cache.GeoUnit(strings.ToLower(args[3]))
	if err1 != nil {
		return c.writer.WriteError("ERR invalid float value")
	}
	withCoord, withDist, count := parseGeoRadiusOpts(args[4:])
	results, err := ts.geoCache.GeoRadiusByMember(args[0], args[1], radius, unit, withCoord, withDist, count)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	return writeGeoResults(c, results, withDist, withCoord)
}

func parseGeoRadiusOpts(args []string) (withCoord, withDist bool, count int) {
	for i := 0; i < len(args); i++ {
		switch strings.ToUpper(args[i]) {
		case "WITHCOORD":
			withCoord = true
		case "WITHDIST":
			withDist = true
		case "COUNT":
			if i+1 < len(args) {
				count, _ = strconv.Atoi(args[i+1])
				i++
			}
		}
	}
	return
}

func writeGeoResults(c *client, results []cache.GeoSearchResult, withDist, withCoord bool) error {
	c.writer.StartArray(len(results))
	for _, r := range results {
		if withDist || withCoord {
			elemCount := 1
			if withDist {
				elemCount++
			}
			if withCoord {
				elemCount++
			}
			c.writer.StartArray(elemCount)
			c.writer.WriteBulkString(r.Name)
			if withDist {
				c.writer.WriteBulkString(fmt.Sprintf("%.4f", r.Distance))
			}
			if withCoord {
				c.writer.StartArray(2)
				c.writer.WriteBulkString(fmt.Sprintf("%.6f", r.Longitude))
				c.writer.WriteBulkString(fmt.Sprintf("%.6f", r.Latitude))
			}
		} else {
			c.writer.WriteBulkString(r.Name)
		}
	}
	return nil
}

func (ts *TCPServer) cmdEval(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'eval' command")
	}
	numKeys, err := strconv.Atoi(args[1])
	if err != nil {
		return c.writer.WriteError("ERR value is not an integer")
	}
	var keysAndArgs []string
	if len(args) > 2 {
		keysAndArgs = args[2:]
	}
	result, err := ts.scriptEngine.Eval(args[0], numKeys, keysAndArgs)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	return writeScriptResult(c, result)
}

func (ts *TCPServer) cmdEvalSHA(c *client, args []string) error {
	if len(args) < 2 {
		return c.writer.WriteError("ERR wrong number of arguments for 'evalsha' command")
	}
	numKeys, err := strconv.Atoi(args[1])
	if err != nil {
		return c.writer.WriteError("ERR value is not an integer")
	}
	var keysAndArgs []string
	if len(args) > 2 {
		keysAndArgs = args[2:]
	}
	result, err := ts.scriptEngine.EvalSHA(args[0], numKeys, keysAndArgs)
	if err != nil {
		return c.writer.WriteError(err.Error())
	}
	return writeScriptResult(c, result)
}

func (ts *TCPServer) cmdScript(c *client, args []string) error {
	if len(args) < 1 {
		return c.writer.WriteError("ERR wrong number of arguments for 'script' command")
	}
	switch strings.ToUpper(args[0]) {
	case "LOAD":
		if len(args) < 2 {
			return c.writer.WriteError("ERR wrong number of arguments for 'script load' command")
		}
		sha, err := ts.scriptEngine.ScriptLoad(args[1])
		if err != nil {
			return c.writer.WriteError(err.Error())
		}
		return c.writer.WriteBulkString(sha)
	case "EXISTS":
		if len(args) < 2 {
			return c.writer.WriteError("ERR wrong number of arguments for 'script exists' command")
		}
		exists := ts.scriptEngine.ScriptExists(args[1:]...)
		c.writer.StartArray(len(exists))
		for _, e := range exists {
			if e {
				c.writer.WriteInteger(1)
			} else {
				c.writer.WriteInteger(0)
			}
		}
		return nil
	case "FLUSH":
		ts.scriptEngine.ScriptFlush()
		return c.writer.WriteSimpleString("OK")
	default:
		return c.writer.WriteError("ERR unknown subcommand for 'script'")
	}
}

func writeScriptResult(c *client, result any) error {
	if result == nil {
		return c.writer.WriteBulkString("")
	}
	switch v := result.(type) {
	case string:
		return c.writer.WriteBulkString(v)
	case int:
		return c.writer.WriteInteger(int64(v))
	case int64:
		return c.writer.WriteInteger(v)
	case float64:
		return c.writer.WriteBulkString(fmt.Sprintf("%v", v))
	case bool:
		if v {
			return c.writer.WriteInteger(1)
		}
		return c.writer.WriteInteger(0)
	default:
		return c.writer.WriteBulkString(fmt.Sprintf("%v", v))
	}
}

func writeBulkStringArray(c *client, items []string) error {
	c.writer.StartArray(len(items))
	for _, item := range items {
		c.writer.WriteBulkString(item)
	}
	return nil
}

func writeBulkStringOrNil(c *client, val string, found bool) error {
	if !found {
		return c.writer.WriteBulkString("")
	}
	return c.writer.WriteBulkString(val)
}

func writeIntegerOrNil(c *client, val int64, found bool) error {
	if !found {
		return c.writer.WriteBulkString("")
	}
	return c.writer.WriteInteger(val)
}

func writeValueFromCache(c *client, val any, found bool) error {
	if !found {
		return c.writer.WriteBulkString("")
	}
	switch v := val.(type) {
	case string:
		return c.writer.WriteBulkString(v)
	case int64:
		return c.writer.WriteInteger(v)
	case int:
		return c.writer.WriteInteger(int64(v))
	case float64:
		return c.writer.WriteBulkString(fmt.Sprintf("%v", v))
	case bool:
		if v {
			return c.writer.WriteInteger(1)
		}
		return c.writer.WriteInteger(0)
	default:
		return c.writer.WriteBulkString(fmt.Sprintf("%v", v))
	}
}

func writeStringMap(c *client, m map[string]string) error {
	c.writer.StartArray(len(m) * 2)
	for k, v := range m {
		c.writer.WriteBulkString(k)
		c.writer.WriteBulkString(v)
	}
	return nil
}

func writeStringSlice(c *client, items []string) error {
	c.writer.StartArray(len(items))
	for _, item := range items {
		c.writer.WriteBulkString(item)
	}
	return nil
}

func writeAnySlice(c *client, items []any) error {
	c.writer.StartArray(len(items))
	for _, item := range items {
		writeValueFromCache(c, item, true)
	}
	return nil
}

func writeScoredMembers(c *client, members []cache.ScoredMember, withScores bool) error {
	c.writer.StartArray(len(members))
	for _, m := range members {
		c.writer.WriteBulkString(m.Member)
		if withScores {
			c.writer.WriteBulkString(strconv.FormatFloat(m.Score, 'f', -1, 64))
		}
	}
	return nil
}

func writeRespWriterArray(w *resp.Writer, count int) {
	w.StartArray(count)
}
