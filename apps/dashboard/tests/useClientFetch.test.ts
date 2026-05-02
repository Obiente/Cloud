import { beforeEach, describe, expect, it, vi } from "vitest";
import { isRef, nextTick, ref, type Ref } from "vue";
import { useClientFetch } from "../app/composables/useClientFetch";

type AsyncDataResult<T> = {
  data: Ref<T | undefined>;
  refresh: ReturnType<typeof vi.fn>;
};

describe("useClientFetch", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "useAsyncData",
      vi.fn(<T>(_key: string | Ref<string>, _fn: () => Promise<T>, options: { default?: () => T } = {}) => {
        return {
          data: ref(options.default ? options.default() : undefined) as Ref<T | undefined>,
          refresh: vi.fn(),
        } satisfies AsyncDataResult<T>;
      })
    );
  });

  it("clears stale data synchronously when a dynamic key changes", async () => {
    const organizationId = ref("org-a");
    const result = useClientFetch(
      () => `deployments-${organizationId.value}`,
      async () => [{ id: "fresh" }],
      { default: () => [] as Array<{ id: string }> }
    );

    result.data.value = [{ id: "deployment-from-org-a" }];

    organizationId.value = "org-b";

    expect(result.data.value).toEqual([]);
    await nextTick();
    expect(result.data.value).toEqual([]);
  });

  it("does not clear data when preserveDataOnKeyChange is enabled", () => {
    const organizationId = ref("org-a");
    const result = useClientFetch(
      () => `deployments-${organizationId.value}`,
      async () => [{ id: "fresh" }],
      {
        default: () => [] as Array<{ id: string }>,
        preserveDataOnKeyChange: true,
      }
    );

    result.data.value = [{ id: "deployment-from-org-a" }];

    organizationId.value = "org-b";

    expect(result.data.value).toEqual([{ id: "deployment-from-org-a" }]);
  });

  it("passes a reactive async data key through to Nuxt", () => {
    const organizationId = ref("org-a");
    useClientFetch(() => `deployments-${organizationId.value}`, async () => []);

    const mockedUseAsyncData = vi.mocked((globalThis as typeof globalThis & { useAsyncData: unknown }).useAsyncData);
    const key = mockedUseAsyncData.mock.calls[0]?.[0];

    expect(isRef(key)).toBe(true);
    expect((key as Ref<string>).value).toBe("deployments-org-a");
    organizationId.value = "org-b";
    expect((key as Ref<string>).value).toBe("deployments-org-b");
  });
});
