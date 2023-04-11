rule test {
  emit "a", 100;
  emit "a", "Hello World";
}

rule "a" if $ == "Hello World" {
  println("rule hello world");
}

rule "a" if $ == 100 {
  println("Val: ", $);
  println("rule 100");
}
