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
