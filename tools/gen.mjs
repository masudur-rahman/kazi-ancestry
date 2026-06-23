// Generates the flat people dataset from a curated family tree.
// Output:
//   web/family.json  -> the single data source the app/server loads.
//
// The RAW tree below is a FICTIONAL sample for the public repo. Real family
// data is never committed; supply it at runtime (web/family.local.json, or the
// backend DB) — both override this sample.
//
// Run:  node tools/gen.mjs
import { writeFileSync } from "node:fs";

// Curated tree. Fields: n = name, origin = place of origin, alias = also-known-as,
// note = free text, tags = string[] (e.g. "died_young" renders a ✻ mark).
const RAW = { n: "রহিম মিয়া", origin: "কাল্পনিকপুর", k: [
  { n: "করিম মিয়া", k: [
    { n: "জামাল", k: [ { n: "নাদিয়া" }, { n: "সাকিব" } ] },
    { n: "কামাল", alias: "কামু" },
    { n: "সুমি", tags: ["died_young"] },
    { n: "সালমা", note: "একটি কাল্পনিক টীকা।", k: [ { n: "রুবেল" }, { n: "রিনা" } ] },
    { n: "জসিম", k: [ { n: "তানভীর" }, { n: "মিতু" } ] }
  ] },
  { n: "হাসান মিয়া", k: [ { n: "ফারুক" }, { n: "নীলা" } ] }
] };

const out = [];
let id = 0;
const walk = (node, parentId) => {
  const pid = id++;
  out.push({
    id: pid, name: node.n, parentId,
    origin: node.origin || "", alias: node.alias || "", spouse: "",
    birth: "", death: "", note: node.note || "",
    tags: Array.isArray(node.tags) ? node.tags : [],
  });
  (node.k || []).forEach((ch) => walk(ch, pid));
};
walk(RAW, null);

writeFileSync("web/family.json", JSON.stringify(out, null, 2) + "\n");
console.log("wrote web/family.json (" + out.length + " people)");
