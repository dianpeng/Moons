fn testAnchor() {
  {
    let m = [0, 1];
    assert::eq(m | q::first, 0);
    assert::eq(m | q::last, 1);
  }

  {
    let m = (0, 1);
    assert::eq(m | q::first, 0);
    assert::eq(m | q::last, 1);
  }

  {
    let m = [0, 1];
    assert::eq(m | q::rest, [1]);
  }
  {
    let m = (0, 1);
    assert::eq(m | q::rest, [1]);
  }
  {
    let m = [];
    assert::eq(m | q::rest, []);
  }

  {
    assert::throw(
      fn () {
        let _ = {} | q::first;
      }
    );

    assert::throw(
      fn () {
        let _ = {} | q::last;
      }
    );

    assert::throw(
      fn() {
        let _ = [] | q::first;
      }
    );

    assert::throw(
      fn() {
        let _ = [] | q::last;
      }
    );
  }
}

fn testSelect() {
  {
    let m = [];
    assert::eq(m | q::select(1, 2, 3), []);
  }
  assert::eq([1] | q::select(0), [1]);
  assert::eq([1, 2, 3] | q::select(1, 2), [2, 3]);
  assert::eq([1, 2, 3] | q::select(0, 2), [1, 3]);
  assert::eq([1, 2, 3] | q::select(0, 1), [1, 2]);
  assert::eq(
    {
      "a": 1,
      "b": 2,
      "c": 3
    } | q::select("a", "b"), {"a": 1, "b": 2});

  assert::eq(
    {
      "a": 1,
      "b": 2,
      "c": 3
    } | q::select("a", "c"), {"a": 1, "c": 3});

  assert::eq(
    {
      "a": 1,
      "b": 2,
      "c": 3
    } | q::select("a"), {"a": 1});
}

fn testMap() {
  {
    let m = [1, 2, 3, 4, 5];
    let output = q::map(
      m,
      fn (key, value) {
        if key % 2 == 0 {
          return ("even", value);
        } else {
          return ("odd", value);
        }
      }
    );
    assert::eq(output.even, [1, 3, 5]);
    assert::eq(output.odd, [2, 4]);
  }
  {
    let m = {
      "a0" : 1,
      "b0" : 2,
      "a1" : 3,
      "b1" : 4
    };

    let output = q::map(
      m,
      fn (key, value) {
        if str::has_prefix(key, "a") {
          return ("aprefix", value);
        } else {
          return ("bprefix", value);
        }
      }
    );

    assert::eq(output.aprefix:length(), 2);
    assert::eq(output.bprefix:length(), 2);
  }
}

fn testSlice() {
  {
    let m = [1, 2, 3];
    assert::eq(m | q::slice(1), [2, 3]);
    assert::eq(m | q::slice(0, 1), [1]);
    assert::eq(m | q::slice(0, 100, 2), [1, 3]);
  }
}

fn testQ() {
  {
    let m = [1, 2, 3];
    assert::eq(m | q::filter(
      fn(k, v): v % 2 == 0
    ), [2]);
  }
  assert::eq([1, 2, 3, 4] | q::filter(
    fn(k, v): v % 2 != 0
  ), [1, 3]);
  assert::eq(
    {
      "a": 1,
      "b": 2,
      "c": 3,
      "d": 4
    } | q::filter(fn(k, v): v % 2 == 0),
    {
      "b": 2,
      "d": 4
    }
  );
}

fn testAgg() {
  {
    let m = [1, 2];
    assert::eq(m | q::max, 2);
    assert::eq(m | q::min, 1);
    assert::eq([] | q::max, null);
    assert::eq([] | q::min, null);
  }
  {
    let m = [1.0, 2.0];
    assert::eq(m | q::max, 2.0);
    assert::eq(m | q::min, 1.0);
    assert::eq([] | q::max, null);
    assert::eq([] | q::min, null);
  }
  {
    let m = [1, 2];
    assert::eq(m | q::sum, 3);
    assert::eq([] | q::sum, null);
  }
  {
    assert::eq([] | q::count, 0);
    assert::eq([1, 2, 3] | q::count, 3);
  }
}

test {
  testAnchor();

  // projection
  testSelect();
  testMap();
  testSlice();
  testQ();

  // agg
  testAgg();
}
