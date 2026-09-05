// The same cases in JavaScript, for bun. It writes 1 CSV line for each case:
// the name, the runtime, the nanoseconds of 1 operation, and the count of the
// operations.
import { readFileSync } from "node:fs";

const budgetMs = 300;
const piece = "0123456789";

function measure(name, f) {
  f();
  const t0 = performance.now();
  let n = 0;
  while (performance.now() - t0 < budgetMs) {
    f();
    n++;
  }
  const el = performance.now() - t0;
  console.log(`${name},bun,${Math.round((el * 1e6) / n)},${n}`);
}

class Activity {
  constructor(m) {
    this.name = m.name;
    this.description = m.description;
    this.locationName = m.locationName;
    this.duration = m.duration;
    this.timeOfDay = m.timeOfDay;
    this.familyFriendly = m.familyFriendly;
    this.price = m.price;
    this.destinationRef = m.destinationRef;
    this.ref = m.ref;
    this.imageUrl = m.imageUrl;
  }
}

const path = process.argv[2];
if (!path) {
  console.error("usage: bench.js <path of activities.json>");
  process.exit(2);
}
const text = readFileSync(path, "utf8");
let typed = JSON.parse(text).map((m) => new Activity(m));

measure("json.decode.any", () => JSON.parse(text).length);
measure("json.decode.typed", () => {
  typed = JSON.parse(text).map((m) => new Activity(m));
  return typed.length;
});
measure("json.encode.typed", () => JSON.stringify(typed).length);
measure("str.concat.5k", () => {
  let s = "";
  for (let i = 0; i < 5000; i++) s += piece;
  return s.length;
});
measure("str.builder.200k", () => {
  const parts = [];
  for (let i = 0; i < 200000; i++) parts.push(piece);
  return parts.join("").length;
});
measure("str.substr.100k", () => {
  const parts = [];
  const span = text.length - 21;
  for (let i = 0; i < 100000; i++) {
    const off = (i * 977) % span;
    parts.push(text.substring(off, off + 20));
  }
  return parts.join("").length;
});

// The scalar cases. A JS number holds 53 bits of an integer, thus int.hash.1m
// reads a BigInt. That case measures a BigInt against a native 64 bit integer,
// and the 2 numbers do not compare.
const numbers = ["0", "7", "42", "1234", "99999", "2147483647", "-58", "300"];
const decimals = ["0.5", "3.14159", "-2.5", "1e3", "12345.6789", "0.0001"];

class Point {
  constructor(x, y, w, h) {
    this.x = x;
    this.y = y;
    this.w = w;
    this.h = h;
  }
}

measure("int.sum.1m", () => {
  let s = 0;
  for (let i = 0; i < 1000000; i++) s += i;
  return s;
});
measure("int.hash.1m", () => {
  let h = 0x1000193n;
  const m = 0x100000001b3n;
  const mask = 0xffffffffffffffffn;
  for (let i = 0; i < 1000000; i++) h = ((h ^ BigInt(i)) * m) & mask;
  return Number(h & 0xffffn);
});
measure("int.div.1m", () => {
  let s = 0;
  for (let i = 1; i <= 1000000; i++) s += Math.trunc(1000003 / i) + (1000003 % i);
  return s;
});
measure("int.parse.100k", () => {
  let s = 0;
  for (let i = 0; i < 100000; i++) s += parseInt(numbers[i % numbers.length], 10);
  return s;
});
measure("int.format.100k", () => {
  let s = 0;
  for (let i = 0; i < 100000; i++) s += String(i * 7919).length;
  return s;
});
measure("float.sum.1m", () => {
  let s = 0;
  for (let i = 0; i < 1000000; i++) s += i * 0.5;
  return Math.trunc(s);
});
measure("float.math.1m", () => {
  let s = 0;
  for (let i = 0; i < 1000000; i++) {
    const x = i * 0.001;
    s += Math.sqrt(x) * 1.5 + x * x;
  }
  return Math.trunc(s);
});
measure("float.parse.100k", () => {
  let s = 0;
  for (let i = 0; i < 100000; i++) {
    s += Math.trunc(parseFloat(decimals[i % decimals.length]));
  }
  return s;
});
measure("float.format.100k", () => {
  let s = 0;
  for (let i = 0; i < 100000; i++) s += (i * 1.5).toFixed(3).length;
  return s;
});
measure("slice.append.1m", () => {
  const s = [];
  for (let i = 0; i < 1000000; i++) s.push(i);
  return s.length;
});
// Math.imul gives the low 32 bits of the product. A plain multiply would lose
// the low bits above 2^53, thus bun would sort a different sequence.
measure("slice.sort.100k", () => {
  const s = new Array(100000);
  let x = 1;
  for (let i = 0; i < 100000; i++) {
    x = (Math.imul(x, 1103515245) + 12345) & 0x7fffffff;
    s[i] = x;
  }
  s.sort((a, b) => a - b);
  return s[50000];
});
measure("map.string.100k", () => {
  const m = new Map();
  for (let i = 0; i < 100000; i++) m.set(String(i), i);
  let s = 0;
  for (let i = 0; i < 100000; i++) s += m.get(String(i));
  return s;
});
measure("map.int.100k", () => {
  const m = new Map();
  for (let i = 0; i < 100000; i++) m.set(i * 7919, i);
  let s = 0;
  for (let i = 0; i < 100000; i++) s += m.get(i * 7919);
  return s;
});
measure("struct.alloc.1m", () => {
  let s = 0;
  for (let i = 0; i < 1000000; i++) {
    const p = new Point(i, i * 2, 10, 20);
    s += p.w * p.h + p.x + p.y;
  }
  return Math.trunc(s);
});

// The text work of a client: a format, a split, a join, a replace, and a sort
// of an object by a field.
const words = ["alpha", "beta", "gamma", "delta", "epsilon", "zeta"];

measure("fmt.sprintf.100k", () => {
  let total = 0;
  for (let i = 0; i < 100000; i++) {
    total += `${words[i % words.length]}-${i}-${(i * 1.5).toFixed(2)}`.length;
  }
  return total;
});
measure("str.split.100k", () => {
  const line = words.join(",");
  let total = 0;
  for (let i = 0; i < 100000; i++) total += line.split(",").length;
  return total;
});
measure("str.join.100k", () => {
  let total = 0;
  for (let i = 0; i < 100000; i++) total += words.join(",").length;
  return total;
});
measure("str.replace.100k", () => {
  const line = words.join(",");
  let total = 0;
  for (let i = 0; i < 100000; i++) total += line.replaceAll(",", " | ").length;
  return total;
});
measure("sort.struct.50k", () => {
  const rows = new Array(50000);
  let x = 1;
  for (let i = 0; i < 50000; i++) {
    x = (Math.imul(x, 1103515245) + 12345) & 0x7fffffff;
    rows[i] = { key: x, name: words[i % words.length] };
  }
  rows.sort((a, b) => a.key - b.key);
  return rows[25000].key;
});
