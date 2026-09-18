# sdui-template-compiler (TypeScript)

> English: [README.md](README.md)

[SDUI 템플릿 컴파일러](../README.ko.md)의 레퍼런스 구현이다. ESM과 Node ≥ 20을 사용하며 런타임 의존성은 하나(`js-yaml`)이다.

아직 npm에 게시되지 않았다. 경로 의존성으로 소스에서 사용한다.

## 설치

```sh
cd js-cplr
pnpm install && pnpm build        # emits dist/
```

다른 프로젝트에서 사용하는 방법은 다음과 같다.

```jsonc
// package.json
{ "dependencies": { "sdui-template-compiler": "file:../sdui-template-compiler/js-cplr" } }
```

## CLI

```sh
node dist/cli.js <sdui-root> --out <dir> [--pretty]
# or, once installed as a dependency: pnpm exec sdui-compile <root> --out <dir>
```

`<dir>/manifest.json`과 `<dir>/screens/<id>/<templateVersion>.json`을 기록한다. `--pretty`는 스크린 JSON의 가독성에만 영향을 주며 etag는 항상 압축 형식의 해시이다.

## 라이브러리

```ts
import { ScreenRegistry } from 'sdui-template-compiler';

// Compiles every screen under the root eagerly — throws on any template error.
const registry = ScreenRegistry.build('path/to/sdui-root');

// Newest version threshold ≤ the client app version
// (selection rules: ../docs/integration.md).
const composed = registry.resolve('home', '2.3.0');
if (composed !== undefined) {
  composed.template; // client-ready JSON tree
  composed.etag;     // sha256[:16] of the compact serialization
}

registry.get('home', '2.3.0');   // just the template
registry.ids();                  // all screen ids, in discovery order
registry.versionsOf('home');     // app-version thresholds, ascending
registry.moduleOf('home');       // screen.yaml manifest (versions, params)
```

오류는 적합성 명세에서 고정한 진단 부분 문자열을 메시지에 포함하는 `Error`로 발생한다. 빌드 또는 부팅 경계에서 처리한다.

단일 트리를 조합해야 하는 도구를 위해 하위 수준 구성 요소(`Composer`, `IncludeResolver`, `TokenResolver`, `Versioning`, `Yaml` 등)도 내보내지만, 의도된 진입점은 `ScreenRegistry`이다.

## 테스트

```sh
pnpm test          # conformance suite against ../spec
pnpm typecheck
```
