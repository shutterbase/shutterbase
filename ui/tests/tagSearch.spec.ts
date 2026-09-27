import { describe, it, expect } from "vitest";
import { tagSearchRank, byTagSearchRank, TAG_SEARCH_NO_MATCH } from "src/util/tagSearch";

const car = (name: string, description = "") => ({ name, description });

// real fixtures from the FSG tag list
const ghent = car("car_051|tid_1292|BE Ghent U", "UGent Racing - Ghent University");
const napoli = car("car_122|tid_1051|IT Napoli UNINA", "UniNa Corse - Squadra Corse Federico II - Università degli Studi di Napoli Federico II");
const aachen = car("car_099|tid_258|DE Aachen RWTH", "Ecurie Aix Formula Student Team RWTH Aachen e.V. - Rheinisch-Westfälische Technische Hochschule Aachen");
const karlsruhe = car("car_399|tid_266|DE Karlsruhe UAS", "High Speed Karlsruhe - Hochschule Karlsruhe");
const kaIT = car("car_076|tid_254|DE Karlsruhe IT", "KA-RaceIng Electric - Karlsruhe Institute of Technology");

describe("tagSearchRank (structured triple-tag search)", () => {
  it("finds the zero-padded car number exactly, never via tid substring", () => {
    expect(tagSearchRank(ghent, "51")).toBe(0);
    expect(tagSearchRank(napoli, "51")).toBe(TAG_SEARCH_NO_MATCH);
    expect(tagSearchRank(aachen, "51")).toBe(TAG_SEARCH_NO_MATCH); // tid_258, no 51-prefix
    expect(tagSearchRank(car("car_122|tid_1051|x"), "51")).toBe(TAG_SEARCH_NO_MATCH); // no "1051" hit
  });

  it("accepts the padded form itself and only matches that car", () => {
    expect(tagSearchRank(ghent, "051")).toBe(0);
    const pool = [ghent, napoli, aachen, karlsruhe];
    expect(pool.filter((t) => tagSearchRank(t, "051") >= 0)).toEqual([ghent]);
  });

  it("ranks exact numbers above raw-prefix numbers above text hits", () => {
    const exact = car("car_051|tid_1|X");
    const prefix = car("car_512|tid_2|Y"); // "512" startsWith "51"
    const text = car("car_001|tid_3|X", "team number 510"); // description substring hit
    const sorted = [text, prefix, exact].sort(byTagSearchRank("51"));
    expect(sorted).toEqual([exact, prefix, text]);
    expect(tagSearchRank(prefix, "51")).toBe(1);
    expect(tagSearchRank(text, "51")).toBeGreaterThan(1);
  });

  it("never searches later identifier segments (tid handles)", () => {
    expect(tagSearchRank(aachen, "258")).toBe(TAG_SEARCH_NO_MATCH); // tid_258
    expect(tagSearchRank(car("car_122|tid_1051|x"), "105")).toBe(TAG_SEARCH_NO_MATCH); // tid_1051 prefix
    expect(tagSearchRank(aachen, "390")).toBe(TAG_SEARCH_NO_MATCH); // car_267's tid
    const hypotheticalCar = car("car_258|tid_9|Z");
    expect(tagSearchRank(hypotheticalCar, "258")).toBe(0); // real car numbers still hit exactly
  });

  it("keeps free-text segments and descriptions searchable", () => {
    expect(tagSearchRank(napoli, "napoli")).toBe(2);
    expect(tagSearchRank(karlsruhe, "karlsruhe")).toBe(2);
    expect(tagSearchRank(kaIT, "ka-raceing")).toBe(3); // description-only hit
    expect(tagSearchRank(ghent, "ugent")).toBe(3);
    expect(tagSearchRank(aachen, "rwth")).toBeGreaterThanOrEqual(2);
  });

  it("ANDs multi-word queries across segments and description", () => {
    expect(tagSearchRank(ghent, "51 ghent")).toBeGreaterThan(0);
    expect(tagSearchRank(ghent, "51 ugent")).toBe(3);
    expect(tagSearchRank(ghent, "51 napoli")).toBe(TAG_SEARCH_NO_MATCH);
  });

  it("supports underscore-free input like car51", () => {
    expect(tagSearchRank(ghent, "car51")).toBe(0);
    expect(tagSearchRank(ghent, "car_051")).toBe(0);
    expect(tagSearchRank(ghent, "car5")).toBe(1); // label-carrying prefix, no length guard
  });

  it("does not match tid labels either", () => {
    const pool = [ghent, napoli, aachen];
    expect(pool.every((t) => tagSearchRank(t, "tid") === TAG_SEARCH_NO_MATCH)).toBe(true);
  });

  it("uses displayName when present", () => {
    const tagged = { ...ghent, displayName: "Ghent Rockets" };
    expect(tagSearchRank(tagged, "rockets")).toBe(2);
  });

  it("rejects empty queries and non-matches", () => {
    expect(tagSearchRank(ghent, "")).toBe(TAG_SEARCH_NO_MATCH);
    expect(tagSearchRank(ghent, "   ")).toBe(TAG_SEARCH_NO_MATCH);
    expect(tagSearchRank(ghent, "zzz")).toBe(TAG_SEARCH_NO_MATCH);
    expect(tagSearchRank(napoli, "39")).toBe(TAG_SEARCH_NO_MATCH); // car_267's tid_390 must not surface
  });
});
