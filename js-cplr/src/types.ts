export type JsonPrimitive = boolean | null | number | string;

export type JsonValue = JsonObject | JsonPrimitive | JsonValue[];

export interface JsonObject {
  [key: string]: JsonValue;
}

export interface ComponentFile {
  vars: string[];
  defaults?: Readonly<Record<string, JsonValue>>;
  content: JsonValue;
}

export interface ResolvedInclude {
  filePath: string;
  content: JsonValue;
}

export type IncludeResolver = (
  refPath: string,
  referrerDir: string,
) => ResolvedInclude;

export type TokenResolver = (group: string) => JsonObject;
