local version = redis.call("GET", KEYS[2])
if not version then
    version = -1
end

if tonumber(version) < tonumber(ARGV[2]) then
    redis.call("SET", KEYS[1], ARGV[1], "EX", ARGV[3])
    redis.call("SET", KEYS[2], ARGV[2], "EX", ARGV[3])
    return 1
else
    return 0
end