#version 440
layout(location = 0) in vec2 qt_TexCoord0;
layout(location = 0) out vec4 fragColor;
layout(std140, binding = 0) uniform buf {
    mat4 qt_Matrix;
    float qt_Opacity;
    vec2 glassSize;
    vec2 canvasSize;
    vec2 glassOrigin;
    vec2 pointerUV;
    float glassRadius;
    float hoverAmount;
    float pressedAmount;
    float selectedAmount;
};
layout(binding = 1) uniform sampler2D backdrop;

float roundBox(vec2 p, vec2 halfSize, float r) {
    vec2 q = abs(p) - halfSize + vec2(r);
    return length(max(q, vec2(0.0))) + min(max(q.x, q.y), 0.0) - r;
}
vec3 sampleBackdrop(vec2 pixel) {
    return texture(backdrop, clamp(pixel / max(canvasSize, vec2(1.0)), vec2(0.0), vec2(1.0))).rgb;
}
void main() {
    vec2 local = qt_TexCoord0 * glassSize;
    vec2 p = local - glassSize * 0.5;
    vec2 halfSize = max(glassSize * 0.5 - vec2(0.7), vec2(0.5));
    float r = clamp(glassRadius, 0.0, min(halfSize.x, halfSize.y));
    float d = roundBox(p, halfSize, r);
    float coverage = 1.0 - smoothstep(-0.6, 0.65, d);
    vec2 n = vec2(roundBox(p+vec2(0.65,0.0),halfSize,r)-roundBox(p-vec2(0.65,0.0),halfSize,r),
                  roundBox(p+vec2(0.0,0.65),halfSize,r)-roundBox(p-vec2(0.0,0.65),halfSize,r));
    n /= max(length(n), 0.001);
    float bevel = max(3.0, min(12.0, glassSize.y*0.23));
    float edge = 1.0-smoothstep(0.0,bevel,max(-d,0.0));
    // Edge-local refraction of the actual application backdrop, not a color gradient substitute.
    vec2 pos = glassOrigin + local - n * (edge*edge) * (8.0 + 4.0*hoverAmount + 2.0*pressedAmount);
    float spread = mix(2.0,0.7,edge);
    vec3 rgb = sampleBackdrop(pos)*0.44;
    rgb += sampleBackdrop(pos+vec2(spread,0.0))*0.14;
    rgb += sampleBackdrop(pos-vec2(spread,0.0))*0.14;
    rgb += sampleBackdrop(pos+vec2(0.0,spread))*0.14;
    rgb += sampleBackdrop(pos-vec2(0.0,spread))*0.14;
    rgb = mix(rgb,vec3(1.0),0.23+0.10*selectedAmount+0.08*hoverAmount);
    float rim = exp(-abs(d+1.45)*1.15);
    float facing = dot(n,normalize(vec2(-0.55,-1.0)));
    rgb += vec3((0.045+0.14*max(facing,0.0))*rim);
    rgb -= vec3(0.045*max(-facing,0.0)*edge);
    vec2 pointerDistance = (qt_TexCoord0-pointerUV)*vec2(1.5,0.8);
    rgb += vec3(0.10*hoverAmount*exp(-dot(pointerDistance,pointerDistance)*9.0));
    rgb -= vec3(0.045*pressedAmount);
    fragColor = vec4(clamp(rgb,0.0,1.0)*coverage,coverage)*qt_Opacity;
}
