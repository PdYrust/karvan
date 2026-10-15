import {
  NeutralToneMapping,
  Color,
  DirectionalLight,
  ExtrudeGeometry,
  Group,
  HemisphereLight,
  Mesh,
  MeshPhysicalMaterial,
  MeshStandardMaterial,
  PerspectiveCamera,
  PMREMGenerator,
  Scene,
  Shape,
  SRGBColorSpace,
  Vector2,
  WebGLRenderer,
} from 'three';
import { RoomEnvironment } from 'three/examples/jsm/environments/RoomEnvironment.js';
import { dashRings, outlineRings, starRings, viewBox, type Ring } from '../logo-geometry';

export interface Palette {
  firouzeh: string;
  saffron: string;
  roadBed: string;
}

export interface LogoScene {
  setPalette(p: Palette): void;
  /** Pointer position in -1..1 on both axes, or null when it leaves. */
  setPointer(x: number | null, y: number | null): void;
  resize(width: number, height: number): void;
  /** onReady fires once the device has shown it keeps up; onTooSlow otherwise. */
  start(onReady: () => void, onTooSlow: () => void): void;
  dispose(): void;
}

// SVG y points down; the scene's y points up, centered on the star.
function shapes(rings: readonly Ring[]): Shape[] {
  const half = viewBox / 2;
  return rings.map((ring) => new Shape(ring.map(([x, y]) => new Vector2(x - half, half - y))));
}

function extrude(rings: readonly Ring[], depth: number, bevel: number): ExtrudeGeometry {
  const g = new ExtrudeGeometry(shapes(rings), {
    depth,
    bevelEnabled: bevel > 0,
    bevelThickness: bevel,
    bevelSize: bevel * 0.7,
    bevelSegments: 4,
    curveSegments: 4,
  });
  g.translate(0, 0, -depth / 2);
  return g;
}

const swayPeriod = 14000;
const swayAngle = 0.42;
const tiltMax = 0.22;
// Under 30 frames in the first two seconds means the device cannot keep up.
const probeMs = 2000;
const probeFrames = 30;

export function createLogoScene(canvas: HTMLCanvasElement, palette: Palette): LogoScene {
  const renderer = new WebGLRenderer({
    canvas,
    antialias: true,
    alpha: true,
    powerPreference: 'low-power',
  });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
  renderer.outputColorSpace = SRGBColorSpace;
  // Neutral tone mapping keeps the turquoise close to the flat mark's color.
  renderer.toneMapping = NeutralToneMapping;
  renderer.toneMappingExposure = 0.95;

  const scene = new Scene();
  const pmrem = new PMREMGenerator(renderer);
  const env = pmrem.fromScene(new RoomEnvironment(), 0.04).texture;
  scene.environment = env;
  scene.environmentIntensity = 0.6;

  const camera = new PerspectiveCamera(28, 1, 10, 2000);
  camera.position.set(0, 0, 470);

  // Glazed tile: a clear coat over a slightly rough turquoise body.
  const glaze = new MeshPhysicalMaterial({
    roughness: 0.38,
    clearcoat: 1,
    clearcoatRoughness: 0.08,
  });
  const bed = new MeshStandardMaterial({ roughness: 0.95 });
  const gold = new MeshStandardMaterial({
    metalness: 0.6,
    roughness: 0.3,
    emissiveIntensity: 0.18,
  });

  const tileDepth = 22;
  const group = new Group();
  const tiles = new Mesh(extrude(starRings, tileDepth, 3.5), glaze);
  const roadBed = new Mesh(extrude(outlineRings, 8, 1.5), bed);
  roadBed.position.z = -tileDepth / 2 + 2;
  const dashes = new Mesh(extrude(dashRings, 2, 0.6), gold);
  dashes.position.z = -tileDepth / 2 + 8;
  group.add(roadBed, tiles, dashes);
  group.rotation.x = -0.12;
  scene.add(group);

  // Lights are white; only the ground bounce takes the road-bed color.
  const white = new Color('white');
  const sky = new HemisphereLight(white, white, 0.6);
  const key = new DirectionalLight(white, 1.6);
  key.position.set(-180, 260, 300);
  const rim = new DirectionalLight(white, 0.7);
  rim.position.set(220, -120, -200);
  scene.add(sky, key, rim);

  const pointer = { x: 0, y: 0, tx: 0, ty: 0 };
  let frame = 0;
  let running = false;

  function setPalette(p: Palette) {
    glaze.color = new Color(p.firouzeh);
    gold.color = new Color(p.saffron);
    gold.emissive = new Color(p.saffron);
    bed.color = new Color(p.roadBed);
    sky.groundColor = new Color(p.roadBed);
  }
  setPalette(palette);

  function start(onReady: () => void, onTooSlow: () => void) {
    running = true;
    const t0 = performance.now();
    let frames = 0;
    let probing = true;
    const loop = (now: number) => {
      if (!running) return;
      pointer.x += (pointer.tx - pointer.x) * 0.06;
      pointer.y += (pointer.ty - pointer.y) * 0.06;
      group.rotation.y =
        Math.sin(((now - t0) / swayPeriod) * Math.PI * 2) * swayAngle + pointer.x * tiltMax;
      group.rotation.x = -0.12 - pointer.y * tiltMax;
      renderer.render(scene, camera);
      frames++;
      if (probing && now - t0 > probeMs) {
        probing = false;
        if (frames < probeFrames) {
          running = false;
          onTooSlow();
          return;
        }
        onReady();
      }
      frame = requestAnimationFrame(loop);
    };
    frame = requestAnimationFrame(loop);
  }

  return {
    setPalette,
    setPointer(x, y) {
      pointer.tx = x ?? 0;
      pointer.ty = y ?? 0;
    },
    resize(width, height) {
      renderer.setSize(width, height, false);
      camera.aspect = width / height;
      camera.updateProjectionMatrix();
    },
    start,
    dispose() {
      running = false;
      cancelAnimationFrame(frame);
      for (const m of [tiles, roadBed, dashes]) m.geometry.dispose();
      glaze.dispose();
      bed.dispose();
      gold.dispose();
      env.dispose();
      pmrem.dispose();
      renderer.dispose();
    },
  };
}
