"""
URL Resolver - Extract product info from affiliate URLs.
Uses facebookexternalhit User-Agent to get OG tags/title from Shopee/TikTok.
"""

import re
import httpx
from typing import Optional


def resolve_product_url(url: str) -> dict:
    """Try to extract product name and price from a Shopee/TikTok URL."""
    result = {
        "product_name": "",
        "price": 0,
        "category": "",
        "resolved": False
    }
    
    try:
        client = httpx.Client(
            timeout=10.0,
            follow_redirects=True,
            headers={
                "User-Agent": "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)"
            }
        )
        
        response = client.get(url)
        html = response.text
        
        # Extract title
        title_match = re.search(r'<title>([^<]+)</title>', html)
        if title_match:
            title = title_match.group(1).strip()
            # Clean up title (remove " | Shopee Indonesia", " - TikTok", etc.)
            title = re.sub(r'\s*\|\s*Shopee\s*Indonesia', '', title)
            title = re.sub(r'\s*-\s*TikTok\s*Shop', '', title)
            title = re.sub(r'^Jual\s+', '', title)  # Remove "Jual " prefix
            result["product_name"] = title.strip()
            result["resolved"] = True
        
        # Try to extract price from meta tags
        price_match = re.search(r'<meta[^>]*property="product:price:amount"[^>]*content="([^"]+)"', html)
        if price_match:
            try:
                result["price"] = float(price_match.group(1))
            except ValueError:
                pass
        
        # Try OG description for category hints
        desc_match = re.search(r'<meta[^>]*property="og:description"[^>]*content="([^"]+)"', html)
        if desc_match:
            desc = desc_match.group(1)
            # Simple category detection from description
            categories = {
                "elektronik": ["modem", "wifi", "router", "headset", "earphone", "charger", "kabel", "speaker"],
                "fashion": ["baju", "celana", "dress", "kaos", "jaket", "sepatu", "tas"],
                "kecantikan": ["serum", "skincare", "makeup", "cream", "masker", "sunscreen"],
                "rumah tangga": ["rak", "organizer", "lampu", "dekorasi"],
                "makanan": ["snack", "kopi", "teh", "bumbu"],
            }
            name_lower = result["product_name"].lower()
            for cat, keywords in categories.items():
                if any(kw in name_lower for kw in keywords):
                    result["category"] = cat
                    break
        
        # If no category from desc, try from product name
        if not result["category"] and result["product_name"]:
            name_lower = result["product_name"].lower()
            categories = {
                "electronics": ["modem", "wifi", "router", "headset", "earphone", "charger", "kabel", "speaker", "bluetooth", "usb", "led"],
                "fashion": ["baju", "celana", "dress", "kaos", "jaket", "sepatu", "tas", "hoodie"],
                "beauty": ["serum", "skincare", "makeup", "cream", "masker", "sunscreen", "moisturizer"],
                "home": ["rak", "organizer", "lampu", "dekorasi", "gantungan"],
                "food": ["snack", "kopi", "teh", "bumbu", "mie"],
            }
            for cat, keywords in categories.items():
                if any(kw in name_lower for kw in keywords):
                    result["category"] = cat
                    break
        
        client.close()
        
    except Exception as e:
        # Silently fail - caller will use manual input
        pass
    
    return result
