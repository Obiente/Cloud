import { beforeEach, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { computed, ref, watch } from "vue";
import { useOrganizationsStore } from "../app/stores/organizations";

let app: {isHydrating:boolean};
let ready: Array<()=>void>;
let storage: Map<string,string>;
beforeEach(()=>{
 setActivePinia(createPinia());
 app={isHydrating:true};ready=[];storage=new Map();
 vi.stubGlobal("ref",ref);vi.stubGlobal("computed",computed);vi.stubGlobal("watch",watch);
 vi.stubGlobal("useNuxtApp",()=>app);
 vi.stubGlobal("onNuxtReady",(callback:()=>void)=>ready.push(callback));
 vi.stubGlobal("useCookie",()=>ref("org-a"));
 vi.stubGlobal("window",{addEventListener:vi.fn()});
 vi.stubGlobal("localStorage",{
  getItem:(key:string)=>storage.get(key)??null,
  setItem:(key:string,value:string)=>storage.set(key,value),
  removeItem:(key:string)=>storage.delete(key),
 });
});

it("keeps the SSR organization snapshot until hydration completes",()=>{
 storage.set("obiente_orgs_cache",JSON.stringify([{id:"org-a",name:"Cached Alpha"}]));
 storage.set("selectedOrgId","org-b");
 const store=useOrganizationsStore();
 store.hydrate();store.hydrate();
 expect(store.orgs).toEqual([]);
 expect(ready).toHaveLength(1);
 app.isHydrating=false;ready[0]!();
 expect(store.orgs).toEqual([{id:"org-a",name:"Cached Alpha"}]);
 expect(store.currentOrgId).toBe("org-a");
});

it("switches and resets in memory when browser storage is unavailable",()=>{
 app.isHydrating=false;
 const deny=()=>{throw new Error("storage denied")};
 vi.stubGlobal("localStorage",{getItem:deny,setItem:deny,removeItem:deny});
 const store=useOrganizationsStore();
 expect(()=>store.switchOrganization("org-b")).not.toThrow();
 expect(store.currentOrgId).toBe("org-b");
 expect(()=>store.reset()).not.toThrow();
 expect(store.currentOrgId).toBe("");
});

it("does not replace newer server data with the delayed local cache",()=>{
 storage.set("obiente_orgs_cache",JSON.stringify([{id:"org-a",name:"Old name"}]));
 const store=useOrganizationsStore();
 store.setOrganizations([{id:"org-a",name:"Current name"}]);
 app.isHydrating=false;ready[0]!();
 expect(store.currentOrg?.name).toBe("Current name");
});
