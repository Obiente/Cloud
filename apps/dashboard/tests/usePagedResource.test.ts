import { expect, it, vi } from "vitest";
import { usePagedResource } from "../app/composables/usePagedResource";
it("loads all 128 events once across three pages and blocks overlapping loads", async () => {
  const rows = Array.from({length:128},(_,i)=>({id:String(i)}));
  const fetch = vi.fn(async (offset:number) => ({items:rows.slice(offset,offset+50),total:128}));
  const paged = usePagedResource(fetch);
  await paged.refresh();
  await Promise.all([paged.loadMore(),paged.loadMore()]);
  await paged.loadMore();
  await paged.loadMore();
  expect(paged.items.value).toHaveLength(128);
  expect(new Set(paged.items.value.map(row=>row.id)).size).toBe(128);
  expect(fetch.mock.calls.map(call=>call[0])).toEqual([0,50,100]);
});
it("ignores a stale page after filters reset and retries a failed page at the same offset", async () => {
  let finish!: (value:{items:{id:string}[];total:number})=>void;
  const fetch = vi.fn().mockImplementationOnce(()=>new Promise(resolve=>{finish=resolve;}))
    .mockRejectedValueOnce(new Error("offline")).mockResolvedValue({items:[{id:"new"}],total:1});
  const paged=usePagedResource<{id:string}>(fetch);
  const old=paged.refresh();
  await paged.refresh();
  expect(paged.error.value).toBeTruthy();
  finish({items:[{id:"old"}],total:1}); await old;
  expect(paged.items.value).toEqual([]);
  await paged.loadMore();
  expect(paged.items.value).toEqual([{id:"new"}]);
  expect(fetch.mock.calls.map(call=>call[0])).toEqual([0,0,0]);
});

it("reports inconsistent empty pages and preserves the retry offset", async () => {
  const fetch = vi.fn().mockResolvedValueOnce({items:[],total:10}).mockResolvedValue({items:[{id:"one"}],total:1});
  const paged = usePagedResource<{id:string}>(fetch);
  await paged.refresh();
  expect(paged.error.value?.message).toContain("refresh");
  await paged.loadMore();
  expect(paged.items.value).toEqual([{id:"one"}]);
  expect(fetch.mock.calls.map(call=>call[0])).toEqual([0,0]);
});
