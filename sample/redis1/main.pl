config redis_vhost {
  .name = "redis_test";
  .listener = "test";
}

rule "redis.HGET" if $:length() == 1 {
  println("command info: (", $.command, ", ", $:length(), ", ", $.category, ")");

  let value = $:asString(0);
  println("command HGET ", value);
  conn:writeString("Always HGET yeah!");
}

rule "redis.HGET" if $:length() != 1 {
  println("command info: (", $.command, ", ", $:length(), ", ", $.category, ")");
  conn:writeError("invalid redis.HGET");
}

rule "redis.:hash" {
  println(">>> command info: (", $.command, ", ", $:length(), ", ", $.category, ")");
  conn:writeError("this is just a sample");
}

rule "redis.*" {
  println("command info: (", $.command, ", ", $:length(), ", ", $.category, ")");
  conn:writeString("Hello World");
}
