from datetime import date, timedelta
from pathlib import Path

from reportlab.lib import colors
from reportlab.lib.enums import TA_CENTER
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
from reportlab.lib.units import mm
from reportlab.platypus import (
    PageBreak,
    Paragraph,
    SimpleDocTemplate,
    Spacer,
    Table,
    TableStyle,
)


OUTPUT = Path(__file__).resolve().parents[1] / "output" / "pdf" / "patient_anant_large_record.pdf"
PAGE_WIDTH, PAGE_HEIGHT = A4


def page_background(canvas, document) -> None:
    canvas.saveState()
    canvas.setFillColor(colors.white)
    canvas.rect(0, 0, PAGE_WIDTH, PAGE_HEIGHT, fill=1, stroke=0)
    if document.page > 1:
        canvas.setStrokeColor(colors.HexColor("#CBD5E1"))
        canvas.line(20 * mm, PAGE_HEIGHT - 17 * mm, PAGE_WIDTH - 20 * mm, PAGE_HEIGHT - 17 * mm)
        canvas.setFillColor(colors.HexColor("#475569"))
        canvas.setFont("Helvetica", 8)
        canvas.drawString(20 * mm, PAGE_HEIGHT - 13 * mm, "SYNTHETIC DEMO RECORD - ANANT SHARMA")
        canvas.drawRightString(PAGE_WIDTH - 20 * mm, 12 * mm, f"Page {document.page}")
    canvas.restoreState()


def styled_table(rows, widths):
    table = Table(rows, colWidths=widths, repeatRows=1)
    table.setStyle(
        TableStyle(
            [
                ("BACKGROUND", (0, 0), (-1, 0), colors.HexColor("#0F766E")),
                ("TEXTCOLOR", (0, 0), (-1, 0), colors.white),
                ("FONTNAME", (0, 0), (-1, 0), "Helvetica-Bold"),
                ("BACKGROUND", (0, 1), (-1, -1), colors.HexColor("#F8FAFC")),
                ("TEXTCOLOR", (0, 1), (-1, -1), colors.HexColor("#0F172A")),
                ("FONTNAME", (0, 1), (-1, -1), "Helvetica"),
                ("FONTSIZE", (0, 0), (-1, -1), 9),
                ("GRID", (0, 0), (-1, -1), 0.5, colors.HexColor("#CBD5E1")),
                ("VALIGN", (0, 0), (-1, -1), "TOP"),
                ("TOPPADDING", (0, 0), (-1, -1), 7),
                ("BOTTOMPADDING", (0, 0), (-1, -1), 7),
            ]
        )
    )
    return table


def add_section(story, styles, title, subtitle, paragraphs, table_rows=None):
    story.append(Paragraph(title, styles["SectionTitle"]))
    story.append(Paragraph(subtitle, styles["SectionSubtitle"]))
    story.append(Spacer(1, 5 * mm))
    for paragraph in paragraphs:
        story.append(Paragraph(paragraph, styles["BodyText"])); story.append(Spacer(1, 4 * mm))
    if table_rows:
        story.append(Spacer(1, 2 * mm))
        story.append(styled_table(table_rows, [47 * mm, 105 * mm]))
    story.append(PageBreak())


def build_pdf() -> None:
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    styles = getSampleStyleSheet()
    styles["BodyText"].fontName = "Helvetica"
    styles["BodyText"].fontSize = 10
    styles["BodyText"].leading = 15
    styles["BodyText"].textColor = colors.HexColor("#1E293B")
    styles.add(ParagraphStyle(name="CoverTitle", parent=styles["Title"], fontSize=28, leading=34, textColor=colors.HexColor("#0F766E"), alignment=TA_CENTER))
    styles.add(ParagraphStyle(name="CoverSub", parent=styles["BodyText"], fontSize=13, leading=19, alignment=TA_CENTER, textColor=colors.HexColor("#475569")))
    styles.add(ParagraphStyle(name="SectionTitle", parent=styles["Heading1"], fontSize=22, leading=27, textColor=colors.HexColor("#0F766E"), spaceAfter=5))
    styles.add(ParagraphStyle(name="SectionSubtitle", parent=styles["BodyText"], fontSize=10, leading=14, textColor=colors.HexColor("#64748B")))

    document = SimpleDocTemplate(
        str(OUTPUT), pagesize=A4, rightMargin=22 * mm, leftMargin=22 * mm,
        topMargin=24 * mm, bottomMargin=20 * mm,
        title="Large synthetic longitudinal patient record - Anant Sharma",
        author="rag-document-assistant",
        pageCompression=1,
    )
    story = [
        Spacer(1, 42 * mm),
        Paragraph("Synthetic Longitudinal<br/>Patient Record", styles["CoverTitle"]),
        Spacer(1, 10 * mm),
        Paragraph("Anant Sharma | Record DEMO-ANANT-LARGE-2026", styles["CoverSub"]),
        Spacer(1, 18 * mm),
        Paragraph(
            "A 41-page fictional record created solely for retrieval-augmented generation testing. "
            "Every name, date, result, diagnosis, and instruction in this document is invented. "
            "This document contains no real patient information and is not medical advice.",
            styles["CoverSub"],
        ),
        Spacer(1, 22 * mm),
        styled_table(
            [["Demo field", "Value"], ["Primary condition", "Jaundice associated with fictional hepatitis A"],
             ["Record period", "18 September 2026 to 26 March 2027"],
             ["Pages", "Cover, clinical sections, 28 monitoring entries, and appendices"]],
            [45 * mm, 105 * mm],
        ),
        PageBreak(),
    ]

    sections = [
        (
            "1. Patient Identity and Record Scope", "Core demographic and record-control information",
            ["Anant Sharma is a fictional 34-year-old patient used only for this software demonstration. His synthetic record number is DEMO-ANANT-LARGE-2026. The initial visit date is 18 September 2026.",
             "The record follows a fabricated episode of jaundice from presentation through recovery. It intentionally distributes facts across many pages so a RAG system can demonstrate chunking, semantic retrieval, grounded answers, and page citations."],
            [["Field", "Recorded value"], ["Patient", "Anant Sharma"], ["Age", "34 years"], ["Blood group", "B positive"], ["Preferred language", "English and Hindi"]],
        ),
        (
            "2. Alerts and Allergies", "High-priority facts for retrieval testing",
            ["The synthetic chart records an allergy to penicillin. The invented reaction is an itchy raised rash that appeared several hours after a childhood dose. No breathing difficulty or loss of consciousness was reported.",
             "There are no recorded food allergies. The chart also states that medicine choices should be checked against the penicillin allergy before prescribing. This alert is repeated only on this page and in the final summary."],
            [["Alert", "Detail"], ["Medication allergy", "Penicillin"], ["Recorded reaction", "Delayed itchy raised rash"], ["Severity", "Moderate; no fictional history of anaphylaxis"]],
        ),
        (
            "3. Initial Presentation", "Symptoms recorded at the first fictional visit",
            ["Anant presented with yellow discoloration of the eyes and skin, dark urine, fatigue, mild nausea, and reduced appetite. Symptoms began five days before the first visit.",
             "He did not report confusion, fainting, bleeding, severe abdominal pain, or breathing difficulty. Vital signs were stable, and the synthetic examination described mild tenderness without guarding."],
            [["Symptom", "First-visit description"], ["Yellow eyes and skin", "Present"], ["Dark urine", "Present"], ["Fatigue", "Moderate"], ["Appetite", "Reduced"]],
        ),
        (
            "4. Baseline Laboratory Results", "Invented results dated 18 September 2026",
            ["The peak total bilirubin in this fictional record was 8.4 mg/dL. Direct bilirubin was 5.1 mg/dL, ALT was 620 U/L, and AST was 410 U/L. These values are demonstration data, not a clinical reference range.",
             "The complete blood count was described as broadly stable. The purpose of these distinct numbers is to test whether the retrieval system can locate exact facts rather than rely on general medical knowledge."],
            [["Test", "Synthetic result"], ["Total bilirubin", "8.4 mg/dL"], ["Direct bilirubin", "5.1 mg/dL"], ["ALT", "620 U/L"], ["AST", "410 U/L"]],
        ),
        (
            "5. Infectious Disease Testing", "Fabricated diagnostic work-up",
            ["The fictional hepatitis A IgM result was positive. Hepatitis B surface antigen and hepatitis C antibody were recorded as negative. The working diagnosis was jaundice associated with acute hepatitis A.",
             "No real laboratory performed these tests. These deliberately specific results allow questions about the suspected cause of jaundice to be answered from this page."],
            [["Test", "Synthetic result"], ["Hepatitis A IgM", "Positive"], ["Hepatitis B surface antigen", "Negative"], ["Hepatitis C antibody", "Negative"]],
        ),
        (
            "6. Abdominal Ultrasound", "Imaging report dated 20 September 2026",
            ["The synthetic abdominal ultrasound described a mildly enlarged liver with uniform echotexture. The gallbladder contained no stones, and the common bile duct was not dilated.",
             "Most importantly for the demo query, the report found no biliary obstruction. The fictional radiologist was Dr Kavita Rao, and the imaging accession code was US-DEMO-2048."],
            [["Finding", "Report text"], ["Liver", "Mildly enlarged; uniform echotexture"], ["Gallbladder", "No stones"], ["Bile duct", "Not dilated"], ["Obstruction", "No biliary obstruction detected"]],
        ),
        (
            "7. Specialist Review", "Fictional hepatology consultation",
            ["Dr Meera Kapoor reviewed the synthetic record on 21 September 2026. She agreed with the working diagnosis of uncomplicated acute hepatitis A and recommended supportive outpatient management.",
             "The plan emphasized hydration, rest, avoidance of alcohol, and avoidance of unapproved medicines or supplements. Repeat liver function testing was scheduled, with escalation for confusion, bleeding, repeated vomiting, or increasing drowsiness."],
            [["Item", "Recorded detail"], ["Consultant", "Dr Meera Kapoor"], ["Specialty", "Hepatology"], ["Management", "Supportive outpatient care"], ["Next review", "Three days"]],
        ),
        (
            "8. Initial Care Plan", "Actions agreed at the start of monitoring",
            ["Anant was advised to drink adequate fluids, eat small balanced meals as tolerated, and rest during the symptomatic period. Alcohol was to be avoided completely during recovery.",
             "The fictional plan did not include an antibiotic or a specific antiviral medicine. Urgent review was advised for confusion, unusual bleeding, persistent vomiting, severe abdominal pain, fainting, or inability to drink fluids."],
            [["Plan element", "Instruction"], ["Hydration", "Regular oral fluids"], ["Activity", "Rest, then gradual return"], ["Alcohol", "Avoid"], ["Follow-up", "Serial symptoms and liver tests"]],
        ),
    ]

    for title, subtitle, paragraphs, rows in sections:
        add_section(story, styles, title, subtitle, paragraphs, rows)

    start = date(2026, 9, 25)
    symptoms = ["fatigue and reduced appetite", "mild fatigue", "improving energy", "no active symptoms"]
    for index in range(28):
        visit_date = start + timedelta(days=index * 7)
        total_bilirubin = max(0.7, round(7.2 - index * 0.31, 1))
        alt = max(28, 510 - index * 19)
        status = symptoms[min(index // 4, len(symptoms) - 1)]
        activity = "rest at home" if index < 2 else "light daily activity" if index < 6 else "usual activity"
        code = f"MON-{visit_date.strftime('%Y%m%d')}-{100 + index}"
        paragraphs = [
            f"At monitoring entry {index + 1}, dated {visit_date.strftime('%d %B %Y')}, Anant reported {status}. "
            f"The permitted activity level was recorded as {activity}. No confusion, bleeding, or persistent vomiting was documented in this fictional entry.",
            f"The synthetic trend value for total bilirubin was {total_bilirubin:.1f} mg/dL and ALT was {alt} U/L. "
            "The record describes a steady improvement over time; these values exist only to make longitudinal retrieval testable.",
            f"The unique entry reference is {code}. The plan was to continue hydration, avoid alcohol, and attend the next scheduled review. "
            "No new medicine allergy or emergency event was added.",
        ]
        rows = [["Monitoring field", "Value"], ["Entry date", visit_date.strftime("%d %B %Y")],
                ["Total bilirubin", f"{total_bilirubin:.1f} mg/dL"], ["ALT", f"{alt} U/L"],
                ["Symptoms", status], ["Activity", activity], ["Reference", code]]
        add_section(story, styles, f"{9 + index}. Weekly Monitoring Entry {index + 1}", "Synthetic longitudinal follow-up", paragraphs, rows)

    appendices = [
        ("37. Recovery Milestone", "Resolution of the fictional acute episode",
         ["By the 60-day milestone, yellow discoloration and dark urine were recorded as resolved. Appetite and energy had returned to baseline, and Anant had resumed usual daily activity.",
          "The chart retained the penicillin allergy alert. Continued avoidance of alcohol was recommended until the planned clinician review, even though the synthetic laboratory trend had normalized."],
         [["Milestone", "Outcome"], ["Jaundice symptoms", "Resolved"], ["Energy", "Returned to baseline"], ["Activity", "Usual activity resumed"]]),
        ("38. Consolidated Laboratory Trend", "Selected values from the demonstration timeline",
         ["The invented laboratory series shows a declining trend rather than a real clinical dataset. The peak total bilirubin was 8.4 mg/dL on 18 September 2026 and the final listed value was 0.7 mg/dL.",
          "This summary exists to test whether retrieval can distinguish the peak value from the final value and return the correct page citation."],
         [["Date", "Total bilirubin"], ["18 September 2026", "8.4 mg/dL (peak)"], ["16 October 2026", "6.0 mg/dL"], ["27 November 2026", "4.1 mg/dL"], ["26 March 2027", "0.7 mg/dL (final listed)"]]),
        ("39. Final Follow-up and Discharge", "Closing fictional review dated 26 March 2027",
         ["At the final follow-up, Anant was described as well, with no jaundice, dark urine, nausea, or unusual fatigue. The synthetic episode was marked resolved, and routine follow-up was advised.",
          "The discharge confirmation phrase is SUNFLOWER-4827. This unusual phrase is intentionally included as a precise retrieval target for the RAG demonstration."],
         [["Field", "Closing value"], ["Episode status", "Resolved"], ["Final review date", "26 March 2027"], ["Discharge phrase", "SUNFLOWER-4827"]]),
        ("40. Final Clinical Summary and Demo Questions", "Key facts and suggested retrieval checks",
         ["Anant Sharma is a fictional 34-year-old patient whose demonstration record describes jaundice associated with acute hepatitis A. Peak total bilirubin was 8.4 mg/dL. Ultrasound found no biliary obstruction. Dr Meera Kapoor provided the specialist review.",
          "The chart records a penicillin allergy with a delayed itchy rash. The episode was marked resolved on 26 March 2027, and the discharge confirmation phrase was SUNFLOWER-4827.",
          "Suggested questions: What medication is Anant allergic to? What was his peak bilirubin? Did ultrasound show an obstruction? Who reviewed him? What was the discharge confirmation phrase? What date was the episode marked resolved?"],
         [["Reminder", "This entire document is synthetic demo data."], ["Purpose", "RAG ingestion, retrieval, generation, and citation testing"]]),
    ]
    for title, subtitle, paragraphs, rows in appendices:
        add_section(story, styles, title, subtitle, paragraphs, rows)

    # The final PageBreak may create a blank page in some renderers, so remove it.
    if story and isinstance(story[-1], PageBreak):
        story.pop()
    document.build(story, onFirstPage=page_background, onLaterPages=page_background)


if __name__ == "__main__":
    build_pdf()
