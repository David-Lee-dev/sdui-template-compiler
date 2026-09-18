import type { ScreenModule, ScreenVersionAssets } from './screen-manifest.js';

interface SemanticVersion {
  major: number;
  minor: number;
  patch: number;
  source: string;
}

export class Versioning {
  /**
   * Resolves the assets declared by the newest supported app-version threshold.
   *
   * @param module - Screen module containing the version map
   * @param appVersion - Optional client application version
   * @returns Template asset version for the threshold
   * @throws When the application version is malformed
   */
  static resolve(
    module: ScreenModule,
    appVersion?: string,
  ): ScreenVersionAssets {
    const versions = Object.keys(module.versions)
      .map((version) => Versioning.parse(version))
      .sort((left, right) => Versioning.compare(left, right));
    if (appVersion === undefined || appVersion.length === 0) {
      return module.versions[versions[0].source];
    }

    const client = Versioning.parse(appVersion);
    const compatible = versions.filter(
      (version) => Versioning.compare(version, client) <= 0,
    );
    const threshold = (compatible.at(-1) ?? versions[0]).source;
    return module.versions[threshold];
  }

  /**
   * Compares two strict major.minor.patch semantic versions.
   *
   * @param left - Left semantic version
   * @param right - Right semantic version
   * @returns A negative, zero, or positive ordering value
   */
  static compareSemver(left: string, right: string): number {
    return Versioning.compare(Versioning.parse(left), Versioning.parse(right));
  }

  private static parse(version: string): SemanticVersion {
    const match = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.exec(version);
    if (match === null) {
      throw new Error(`Invalid semantic version: ${version}`);
    }

    return {
      major: Number(match[1]),
      minor: Number(match[2]),
      patch: Number(match[3]),
      source: version,
    };
  }

  private static compare(
    left: SemanticVersion,
    right: SemanticVersion,
  ): number {
    return (
      left.major - right.major ||
      left.minor - right.minor ||
      left.patch - right.patch
    );
  }
}
