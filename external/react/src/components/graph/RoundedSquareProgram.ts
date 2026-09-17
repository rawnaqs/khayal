import type { Attributes } from 'graphology-types'
import { NodeProgram } from 'sigma/rendering'
import type { ProgramInfo } from 'sigma/rendering'
import type { NodeDisplayData, RenderParams } from 'sigma/types'
import { floatColor } from 'sigma/utils'

// A point-sprite node program (like sigma's NodePointProgram) whose
// fragment shader carves a rounded square instead of a disc. We use
// gl.POINTS rather than triangle geometry so the corners are never
// clipped by the covering triangle, and so the radius is a single knob.
const VERTEX_SHADER_SOURCE = /* glsl */ `
attribute vec4 a_id;
attribute vec4 a_color;
attribute vec2 a_position;
attribute float a_size;

uniform float u_sizeRatio;
uniform float u_pixelRatio;
uniform mat3 u_matrix;

varying vec4 v_color;
varying float v_border;

const float bias = 255.0 / 254.0;

void main() {
  gl_Position = vec4((u_matrix * vec3(a_position, 1)).xy, 0, 1);

  gl_PointSize = a_size / u_sizeRatio * u_pixelRatio * 2.0;
  v_border = (0.5 / a_size) * u_sizeRatio;

  #ifdef PICKING_MODE
  v_color = a_id;
  #else
  v_color = a_color;
  #endif

  v_color.a *= bias;
}
`

// `CORNER_RADIUS` is in sprite units: 0 is a hard square, 0.5 a circle.
// 0.24 keeps a clearly square silhouette with soft corners.
const CORNER_RADIUS = 0.24

const FRAGMENT_SHADER_SOURCE = /* glsl */ `
precision mediump float;

varying vec4 v_color;
varying float v_border;

const vec4 transparent = vec4(0.0, 0.0, 0.0, 0.0);
const float radius = ${CORNER_RADIUS.toFixed(3)};

void main(void) {
  // signed distance to a rounded box centred in the point sprite
  vec2 p = abs(gl_PointCoord - vec2(0.5)) - (vec2(0.5) - vec2(radius));
  float dist = radius - (length(max(p, 0.0)) + min(max(p.x, p.y), 0.0));

  #ifdef PICKING_MODE
  if (dist > v_border)
    gl_FragColor = v_color;
  else
    gl_FragColor = transparent;
  #else
  float t = 0.0;
  if (dist > v_border)
    t = 1.0;
  else if (dist > 0.0)
    t = dist / v_border;

  gl_FragColor = mix(transparent, v_color, t);
  #endif
}
`

const UNIFORMS = ['u_sizeRatio', 'u_pixelRatio', 'u_matrix'] as const

export class RoundedSquareProgram<
  N extends Attributes = Attributes,
  E extends Attributes = Attributes,
  G extends Attributes = Attributes,
> extends NodeProgram<(typeof UNIFORMS)[number], N, E, G> {
  getDefinition() {
    return {
      VERTICES: 1,
      VERTEX_SHADER_SOURCE,
      FRAGMENT_SHADER_SOURCE,
      METHOD: WebGLRenderingContext.POINTS,
      UNIFORMS,
      ATTRIBUTES: [
        { name: 'a_position', size: 2, type: WebGLRenderingContext.FLOAT },
        { name: 'a_size', size: 1, type: WebGLRenderingContext.FLOAT },
        { name: 'a_color', size: 4, type: WebGLRenderingContext.UNSIGNED_BYTE, normalized: true },
        { name: 'a_id', size: 4, type: WebGLRenderingContext.UNSIGNED_BYTE, normalized: true },
      ],
    }
  }

  processVisibleItem(nodeIndex: number, startIndex: number, data: NodeDisplayData) {
    const array = this.array
    array[startIndex++] = data.x
    array[startIndex++] = data.y
    array[startIndex++] = data.size
    array[startIndex++] = floatColor(data.color)
    array[startIndex++] = nodeIndex
  }

  setUniforms(params: RenderParams, { gl, uniformLocations }: ProgramInfo<(typeof UNIFORMS)[number]>) {
    gl.uniform1f(uniformLocations.u_pixelRatio, params.pixelRatio)
    gl.uniform1f(uniformLocations.u_sizeRatio, params.sizeRatio)
    gl.uniformMatrix3fv(uniformLocations.u_matrix, false, params.matrix)
  }
}
