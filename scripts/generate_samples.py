from pathlib import Path

from reportlab.lib.pagesizes import A4
from reportlab.lib import colors
from reportlab.lib.enums import TA_CENTER
from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
from reportlab.lib.units import mm
from reportlab.platypus import Paragraph, SimpleDocTemplate, Spacer, Table, TableStyle


OUTPUT_DIR = Path(__file__).resolve().parents[1] / "output" / "pdf"


def paint_page_background(canvas, document) -> None:
    canvas.saveState()
    canvas.setFillColor(colors.white)
    canvas.rect(0, 0, A4[0], A4[1], fill=1, stroke=0)
    canvas.restoreState()


def create_patient_pdf(filename: str, record: dict[str, str]) -> None:
    output = OUTPUT_DIR / filename
    styles = getSampleStyleSheet()
    styles.add(
        ParagraphStyle(
            name="DemoLabel",
            parent=styles["BodyText"],
            alignment=TA_CENTER,
            textColor=colors.HexColor("#475569"),
            fontSize=9,
            leading=12,
        )
    )
    document = SimpleDocTemplate(
        str(output),
        pagesize=A4,
        rightMargin=24 * mm,
        leftMargin=24 * mm,
        topMargin=24 * mm,
        bottomMargin=24 * mm,
        title=f"Synthetic patient record - {record['patient']}",
        author="rag-document-assistant",
    )
    details = [
        ["Patient", record["patient"], "Record ID", record["record_id"]],
        ["Age", record["age"], "Visit date", record["visit_date"]],
        ["Diagnosis", record["diagnosis"], "Status", record["status"]],
    ]
    table = Table(details, colWidths=[27 * mm, 48 * mm, 28 * mm, 48 * mm])
    table.setStyle(
        TableStyle(
            [
                ("BACKGROUND", (0, 0), (0, -1), colors.HexColor("#E2E8F0")),
                ("BACKGROUND", (2, 0), (2, -1), colors.HexColor("#E2E8F0")),
                ("TEXTCOLOR", (0, 0), (-1, -1), colors.HexColor("#0F172A")),
                ("FONTNAME", (0, 0), (-1, -1), "Helvetica"),
                ("FONTNAME", (0, 0), (0, -1), "Helvetica-Bold"),
                ("FONTNAME", (2, 0), (2, -1), "Helvetica-Bold"),
                ("FONTSIZE", (0, 0), (-1, -1), 9),
                ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
                ("GRID", (0, 0), (-1, -1), 0.5, colors.HexColor("#CBD5E1")),
                ("TOPPADDING", (0, 0), (-1, -1), 7),
                ("BOTTOMPADDING", (0, 0), (-1, -1), 7),
            ]
        )
    )
    story = [
        Paragraph("Synthetic Patient Record", styles["Title"]),
        Spacer(1, 3 * mm),
        Paragraph(
            "DEMO DATA - This fictional record is only for testing the RAG application. "
            "It does not contain real patient information and is not medical advice.",
            styles["DemoLabel"],
        ),
        Spacer(1, 8 * mm),
        table,
        Spacer(1, 8 * mm),
        Paragraph("Clinical Summary", styles["Heading2"]),
        Paragraph(record["summary"], styles["BodyText"]),
        Spacer(1, 5 * mm),
        Paragraph("Reported Symptoms", styles["Heading2"]),
        Paragraph(record["symptoms"], styles["BodyText"]),
        Spacer(1, 5 * mm),
        Paragraph("Investigations", styles["Heading2"]),
        Paragraph(record["investigations"], styles["BodyText"]),
        Spacer(1, 5 * mm),
        Paragraph("Care Plan", styles["Heading2"]),
        Paragraph(record["plan"], styles["BodyText"]),
    ]
    document.build(
        story,
        onFirstPage=paint_page_background,
        onLaterPages=paint_page_background,
    )


def main() -> None:
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    create_patient_pdf(
        "patient_anant.pdf",
        {
            "patient": "Anant Sharma",
            "record_id": "DEMO-ANANT-001",
            "age": "34 years",
            "visit_date": "18 September 2026",
            "diagnosis": "Jaundice",
            "status": "Stable; outpatient follow-up",
            "summary": (
                "Anant presented with yellowing of the eyes and skin, tiredness, and reduced appetite. "
                "The clinician recorded jaundice as the working diagnosis and advised follow-up while "
                "the underlying cause is evaluated."
            ),
            "symptoms": (
                "Yellow discoloration of the eyes, dark urine, fatigue, mild nausea, and reduced appetite. "
                "No confusion or severe abdominal pain was reported in this fictional visit."
            ),
            "investigations": (
                "Liver function tests showed elevated bilirubin. An abdominal ultrasound and repeat liver "
                "function tests were requested for the follow-up visit."
            ),
            "plan": (
                "Maintain hydration, rest, avoid alcohol, and take only clinician-approved medicines. "
                "Return for review in three days, or seek urgent care for worsening pain, repeated vomiting, "
                "bleeding, confusion, or increasing drowsiness."
            ),
        },
    )
    create_patient_pdf(
        "patient_vibhor.pdf",
        {
            "patient": "Vibhor Mehta",
            "record_id": "DEMO-VIBHOR-002",
            "age": "29 years",
            "visit_date": "19 September 2026",
            "diagnosis": "Malaria",
            "status": "Stable; treatment started",
            "summary": (
                "Vibhor presented with intermittent high fever, chills, sweating, headache, and body aches. "
                "A malaria test was positive, and the clinician recorded uncomplicated malaria as the diagnosis."
            ),
            "symptoms": (
                "Episodes of fever with chills followed by sweating, headache, muscle aches, weakness, and mild nausea. "
                "No breathing difficulty, seizure, or altered consciousness was reported in this fictional visit."
            ),
            "investigations": (
                "The malaria rapid diagnostic test was positive. A complete blood count was requested to monitor "
                "haemoglobin and platelet levels during recovery."
            ),
            "plan": (
                "Take the prescribed antimalarial course exactly as directed, drink adequate fluids, and rest. "
                "Follow up in two days. Seek urgent care for persistent vomiting, breathing difficulty, seizure, "
                "confusion, fainting, or inability to drink fluids."
            ),
        },
    )


if __name__ == "__main__":
    main()
