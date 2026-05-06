"""
Threads Affiliate AI Content Generation Service
FastAPI app that generates organic Threads posts using AI.
"""

import os
from typing import Optional

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from dotenv import load_dotenv

from generator import ContentGenerator
from url_resolver import resolve_product_url

load_dotenv()

app = FastAPI(
    title="Threads Affiliate AI Service",
    description="AI content generation for organic Threads affiliate posts",
    version="1.0.0"
)

# Initialize generator
generator = ContentGenerator()


# Request/Response models
class GenerateRequest(BaseModel):
    product_name: str
    price: float = 0
    category: str = ""
    platform: str = "shopee"  # shopee, tiktok, unknown
    persona: str = "honest_friend"
    format: str = "single"  # single, thread, hot_take, question, story
    link_placement: str = "direct"  # direct, reply_drop, bio, question_trigger
    short_url: str = ""


class GenerateResponse(BaseModel):
    content: str
    hashtags: list[str]
    persona: str
    format: str
    link_placement: str
    estimated_engagement: str


class BatchGenerateRequest(BaseModel):
    products: list[GenerateRequest]


class BatchGenerateResponse(BaseModel):
    results: list[GenerateResponse]
    count: int


class PersonaInfo(BaseModel):
    id: str
    name: str


# Endpoints
@app.get("/health")
async def health():
    return {
        "status": "ok",
        "service": "ai-content-generator",
        "model": os.getenv("AI_MODEL", "claude-sonnet-4"),
        "personas_available": len(generator.list_personas())
    }


@app.post("/generate", response_model=GenerateResponse)
async def generate_post(req: GenerateRequest):
    """Generate a single organic Threads post."""
    try:
        result = generator.generate_post(
            product_name=req.product_name,
            price=req.price,
            category=req.category,
            platform=req.platform,
            persona=req.persona,
            format=req.format,
            link_placement=req.link_placement,
            short_url=req.short_url
        )
        return GenerateResponse(**result)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Generation failed: {str(e)}")


@app.post("/generate/batch", response_model=BatchGenerateResponse)
async def generate_batch(req: BatchGenerateRequest):
    """Generate posts for multiple products."""
    if len(req.products) > 20:
        raise HTTPException(status_code=400, detail="Max 20 products per batch")
    
    results = []
    for product in req.products:
        try:
            result = generator.generate_post(
                product_name=product.product_name,
                price=product.price,
                category=product.category,
                platform=product.platform,
                persona=product.persona,
                format=product.format,
                link_placement=product.link_placement,
                short_url=product.short_url
            )
            results.append(GenerateResponse(**result))
        except Exception as e:
            # Skip failed generations, continue with others
            results.append(GenerateResponse(
                content=f"[Generation failed: {str(e)}]",
                hashtags=[],
                persona=product.persona,
                format=product.format,
                link_placement=product.link_placement,
                estimated_engagement="low"
            ))
    
    return BatchGenerateResponse(results=results, count=len(results))


@app.get("/personas", response_model=list[PersonaInfo])
async def list_personas():
    """List available content personas."""
    return generator.list_personas()


@app.get("/formats")
async def list_formats():
    """List available content formats."""
    return {
        "formats": [
            {"id": "single", "description": "Single post (max 500 chars)"},
            {"id": "thread", "description": "Thread of 3-5 connected posts"},
            {"id": "hot_take", "description": "Bold opinion that triggers discussion"},
            {"id": "question", "description": "Question that engages audience"},
            {"id": "story", "description": "Storytelling narrative arc"},
        ],
        "link_placements": [
            {"id": "direct", "description": "Link embedded in post"},
            {"id": "reply_drop", "description": "Link dropped in reply after posting"},
            {"id": "bio", "description": "Reference to bio link"},
            {"id": "question_trigger", "description": "No link, triggers 'where to buy' comments"},
        ]
    }


class ResolveURLRequest(BaseModel):
    url: str


@app.post("/resolve-url")
async def resolve_url(req: ResolveURLRequest):
    """Resolve a Shopee/TikTok URL to extract product info."""
    result = resolve_product_url(req.url)
    return result


if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("AI_SERVICE_PORT", "8081"))
    uvicorn.run(app, host="0.0.0.0", port=port)
