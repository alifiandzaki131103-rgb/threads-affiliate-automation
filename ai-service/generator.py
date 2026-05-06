"""
AI Content Generator for Threads Affiliate posts.
Calls Claude API (or OpenAI-compatible) to generate organic content.
"""

import os
import json
import httpx
from typing import Optional

from prompts import build_prompt, PERSONAS


class ContentGenerator:
    def __init__(self, api_key: Optional[str] = None, api_url: Optional[str] = None):
        self.api_key = api_key or os.getenv("AI_API_KEY", "")
        self.api_url = api_url or os.getenv("AI_API_URL", "http://localhost:1430/v1")
        self.model = os.getenv("AI_MODEL", "claude-sonnet-4")
        self.client = httpx.Client(timeout=60.0)

    def generate_post(
        self,
        product_name: str,
        price: float,
        category: str,
        platform: str,
        persona: str = "honest_friend",
        format: str = "single",
        link_placement: str = "direct",
        short_url: str = ""
    ) -> dict:
        """Generate an organic Threads post using AI."""
        
        # Validate persona
        if persona not in PERSONAS:
            persona = "honest_friend"
        
        # Build prompts
        system_prompt, user_prompt = build_prompt(
            product_name=product_name,
            price=price,
            category=category,
            platform=platform,
            persona=persona,
            format=format,
            link_placement=link_placement,
            short_url=short_url
        )
        
        # Call AI API
        try:
            response = self._call_api(system_prompt, user_prompt)
        except Exception as e:
            # Fallback to template-based generation
            response = self._fallback_generate(
                product_name, price, platform, persona, link_placement, short_url
            )
        
        # Parse response
        content = response.get("content", "")
        
        # Extract hashtags from content
        hashtags = self._extract_hashtags(content)
        
        # Estimate engagement based on format and persona
        engagement = self._estimate_engagement(format, persona)
        
        return {
            "content": content,
            "hashtags": hashtags,
            "persona": persona,
            "format": format,
            "link_placement": link_placement,
            "estimated_engagement": engagement
        }

    def _call_api(self, system_prompt: str, user_prompt: str) -> dict:
        """Call Claude/OpenAI-compatible API."""
        
        url = f"{self.api_url}/chat/completions"
        
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {self.api_key}"
        }
        
        payload = {
            "model": self.model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt}
            ],
            "max_tokens": 1000,
            "temperature": 0.8
        }
        
        response = self.client.post(url, headers=headers, json=payload)
        response.raise_for_status()
        
        data = response.json()
        content = data["choices"][0]["message"]["content"]
        
        return {"content": content}

    def _fallback_generate(
        self, product_name: str, price: float, platform: str,
        persona: str, link_placement: str, short_url: str
    ) -> dict:
        """Fallback template-based generation when API fails."""
        
        platform_text = "Shopee" if platform == "shopee" else "TikTok" if platform == "tiktok" else "online"
        
        templates = {
            "honest_friend": f"Baru coba {product_name} yang {price/1000:.0f}rb dari {platform_text}. Jujur lumayan surprised sama hasilnya buat harga segitu.\n\nPlus: worth the price, works as expected\nMinus: packaging biasa aja\n\nOverall 7/10 sih.",
            "hot_take": f"Unpopular opinion: {product_name} yang cuma {price/1000:.0f}rb itu underrated banget. Orang pada beli yang 3x lipat harganya padahal hasilnya sama aja.",
            "problem_solver": f"Akhirnya nemu solusi yang works: {product_name}. Harga {price/1000:.0f}rb dan so far udah 2 minggu hasilnya keliatan.",
            "curious_explorer": f"Ada yang udah coba {product_name}? Gue baru nemu di {platform_text}, harga {price/1000:.0f}rb tapi reviewnya bagus banget. Penasaran.",
            "lifestyle_sharer": f"Small addition ke routine gue lately: {product_name}. Cuma {price/1000:.0f}rb tapi bikin beda.",
            "comparison_nerd": f"Setelah compare beberapa opsi, {product_name} ({price/1000:.0f}rb) surprisingly menang di value for money."
        }
        
        content = templates.get(persona, templates["honest_friend"])
        
        # Add link based on placement
        if link_placement == "direct" and short_url:
            content += f"\n\nLink: {short_url} (affiliate)"
        elif link_placement == "bio":
            content += "\n\nLink ada di bio kalo mau cek."
        
        content += "\n\n#threads #rekomendasi"
        
        return {"content": content}

    def _extract_hashtags(self, content: str) -> list[str]:
        """Extract hashtags from generated content."""
        words = content.split()
        hashtags = [w for w in words if w.startswith("#")]
        return hashtags[:5]  # Max 5 hashtags

    def _estimate_engagement(self, format: str, persona: str) -> str:
        """Estimate engagement level based on format and persona."""
        high_engagement = {"hot_take", "question"}
        high_personas = {"hot_take", "curious_explorer"}
        
        if format in high_engagement or persona in high_personas:
            return "high"
        elif format == "thread":
            return "medium-high"
        else:
            return "medium"

    def list_personas(self) -> list[dict]:
        """List available personas with descriptions."""
        return [
            {"id": key, "name": val["name"]}
            for key, val in PERSONAS.items()
        ]
